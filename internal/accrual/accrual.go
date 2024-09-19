package accrual

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

var requestPeriod time.Duration // период между итерациями отправки запросов к accrual

func SetRequestPeriod(period time.Duration) {
	requestPeriod = period
}

func GetRequestPeriod() time.Duration {
	return requestPeriod
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
	Accrual float64  `json:"accrual"` // статус расчёта начисления
}

type Retry struct {
	retryPeriod int    // период,в течении которого сервис не должен отправлять запросы к accrual
	message     string // сообщение от accrual
}

type Result struct {
	AccrualData         // данные полученные от accrual
	Retry               // данные полученные от accrual из-за превышения лимита запросов
	err           error // сохраняю внутреннюю ошибку gophermart
	requestStatus int   // код ответа от accrual
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

func waitUntil(date time.Time) {
	duration := time.Until(date) // вычисляем время до указанной даты
	if duration > 0 {
		time.Sleep(duration) // ждем до указанного времени
	}
}

func UpdateAccrualData(ctx context.Context, stor repositories.OrdersInterface, retry repositories.RetryInterface) {
	for {
		// Устанавливаю ожидание, если был превышен лимит обращений к accrual
		waitPeriod, err := retry.GetRetryPeriod(ctx, "accrual")
		if err != nil {
			logger.ServerLog.Error("internal error in GetRetryPeriod", zap.String("error", err.Error()))
		} else {
			waitUntil(waitPeriod)
		}

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
				logger.ServerLog.Error(result.message, zap.String("status", "429"))

				// фиксирую дату и время в UTC, до которого нельзя обращаться к сервису accrual
				// в UTC, потому база данных преобразует время к UTC
				period := time.Now().Add(time.Duration(result.retryPeriod) * time.Second).UTC()
				err := retry.AddRetryPeriod(ctx, "accrual", period)
				logger.ServerLog.Error("internal error", zap.String("error", err.Error()))
			}
			// обработка кода 200 от accrual
			if result.requestStatus == http.StatusNoContent {

			}
		}
		// ожидание между итерациями отправки запросов к accrual
		time.Sleep(GetRequestPeriod() * time.Second)
	}
}
