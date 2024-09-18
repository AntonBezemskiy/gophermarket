package accrual

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/AntonBezemskiy/gophermart/internal/logger"
	"github.com/AntonBezemskiy/gophermart/internal/repositories"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var accrualSystemAddress string

func SetAccrualSystemAddress(adress string) {
	accrualSystemAddress = adress
}

func GetAccrualSystemAddress() string {
	return accrualSystemAddress
}

const (
	REGISTERED = "REGISTERED"
	INVALID    = "INVALID"
	PROCESSING = "PROCESSING"
	PROCESSED  = "PROCESSED"
)

type AccrualData struct {
	Order   string `json:"order"`   // номер заказа
	Status  string `json:"status"`  // статус обработки заказа
	Accrual int64  `json:"accrual"` // статус расчёта начисления
}

type Result struct {
	err           error // сохраняю внутреннюю ошибку gophermart
	AccrualData         // данные полученные от accrual
	requestStatus int   // код ответа от accrual
	retryPeriod   int   // период,в течении которого сервис не должен отправлять запросы к accrual
}

func Sender(jobs <-chan int64, results chan<- Result, client *resty.Client) {
	for {
		// переменная для хранения результата работы Sender
		var responce Result

		num := <-jobs
		url := fmt.Sprintf("%s/%d", GetAccrualSystemAddress(), num)

		resp, err := client.R().
			SetHeader("Content-Type", "text/plain").
			Get(url)

		if err != nil {
			//logger.ServerLog.Error("send request to accrual error", zap.String("address", url), zap.String("error", err.Error()))
			responce.err = fmt.Errorf("send request to accrual error %w", err)
		}

		stat := resp.StatusCode()
		responce.requestStatus = stat
		if stat == http.StatusOK {
			responceAccrual := resp.Body()

			// Десериализую данные полученные от сервиса accrual
			var res AccrualData
			buRes := bytes.NewBuffer(responceAccrual)
			dec := json.NewDecoder(buRes)
			if err := dec.Decode(&res); err != nil {
				//logger.ServerLog.Error("decode data from accrual error", zap.String("error", err.Error()))
				responce.err = fmt.Errorf("decode data from accrual error %w", err)
			}
			responce.AccrualData = res
		}
		// обрабатываю статус: превышено количество запросов к сервису
		if stat == http.StatusTooManyRequests {
			// получаю из accrual период,в течении которого сервис не должен отправлять запросы к accrual
			retryPeriodStr := resp.Header().Get("Retry-After")
			retryPeriod, err := strconv.Atoi(retryPeriodStr)
			if err != nil {
				responce.err = err
			} else {
				responce.retryPeriod = retryPeriod
			}
		}
		// отправка результат
		results <- responce
	}
}

// создаю 10 воркеров, которые отправляют запросы к accrual для расчета баллов лояльности
func Generator(ctx context.Context, stor repositories.OrdersInterface) ([]Result, error) {
	// создаю нового клиента для каждой итерации отправки запросов к accrual
	client := resty.New()

	numbers, err := stor.GetOrdersForAccrual(ctx)
	if err != nil {
		return nil, err
	}

	// пусть будет 10 одновременных запросов к accrual
	const numJobs = 10
	// создаем буферизованный канал для принятия задач в воркер
	jobs := make(chan int64, numJobs)
	// создаем буферизованный канал для отправки результатов
	results := make(chan Result, numJobs)

	for w := 1; w <= 3; w++ {
		go Sender(jobs, results, client)
	}

	for _, num := range numbers {
		jobs <- num
	}
	// закрываю канал
	close(jobs)

	dataFromAccrual := make([]Result, 0)

	for i := 0; i < len(numbers); i++ {
		result := <-results
		dataFromAccrual = append(dataFromAccrual, result)
	}
	return dataFromAccrual, nil
}

func UpdateAccrualData(ctx context.Context, stor repositories.OrdersInterface) {
	for {
		results, err := Generator(ctx, stor)
		logger.ServerLog.Error("get error in UpdateAccrualData", zap.String("error", err.Error()))

		for _, result := range results {
			if result.err != nil {
				logger.ServerLog.Error("internal error", zap.String("error", result.err.Error()))
			}
			// обработка кода 204 от accrual: заказ не зарегистрирован в системе расчёта.
			if result.requestStatus == http.StatusNoContent {
				logger.ServerLog.Error("order is not register in accrual", zap.String("status", "204"))
			}
			// обработка кода 429 от accrual: превышено количество запросов к сервису.
			if result.requestStatus == http.StatusTooManyRequests {
				logger.ServerLog.Error("reached limit to request to accrual", zap.String("status", "429"))
			}
		}
	}
}
