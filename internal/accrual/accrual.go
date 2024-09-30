package accrual

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AntonBezemskiy/gophermart/internal/logger"
	"github.com/AntonBezemskiy/gophermart/internal/repositories"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var (
	accrualSystemAddress string
	requestPeriod        time.Duration = 100 * time.Millisecond // период между итерациями отправки запросов к accrual
	waitOrders           time.Duration = 100 * time.Millisecond // период ожидания поступления заказов в систему
)

func SetAccrualSystemAddress(adress string) {
	accrualSystemAddress = adress
}

func GetAccrualSystemAddress() string {
	return accrualSystemAddress
}

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

type Retry struct {
	retryPeriod int    // период,в течении которого сервис не должен отправлять запросы к accrual
	message     string // сообщение от accrual
}

type Result struct {
	repositories.AccrualData       // данные полученные от accrual
	Retry                          // данные полученные от accrual из-за превышения лимита запросов
	err                      error // сохраняю внутреннюю ошибку gophermart
	requestStatus            int   // код ответа от accrual
}

func Sender(jobs <-chan int64, results chan<- Result, adressAccrual string, client *resty.Client) {
	for num := range jobs {
		// переменная для хранения результата работы Sender
		var responce Result
		url := fmt.Sprintf("%s/api/orders/%d", adressAccrual, num)

		resp, err := client.R().
			SetHeader("Content-Type", "text/plain").
			Get(url)

		if err != nil {
			// отправляю ошибку и выхожу из цикла
			responce.err = fmt.Errorf("send request to accrual error %w", err)
			// отправка результат
			results <- responce
			break
		}

		stat := resp.StatusCode()
		responce.requestStatus = stat
		if stat == http.StatusOK {
			responceAccrual := resp.Body()

			// Десериализую данные полученные от сервиса accrual
			var res repositories.AccrualData
			buRes := bytes.NewBuffer(responceAccrual)
			dec := json.NewDecoder(buRes)
			if err := dec.Decode(&res); err != nil {
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
			// Получаю тело ответа от accrual
			b := resp.Body()
			responce.message = string(b)
		}
		// отправка результата
		results <- responce
	}
}

// создаю 10 воркеров, которые отправляют запросы к accrual для расчета баллов лояльности
func Generator(ctx context.Context, client *resty.Client, stor repositories.OrderManager) ([]Result, error) {
	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		return make([]Result, 0), ctx.Err() 
	default:
		// Контекст ещё не отменен, можно продолжить выполнение
	}

	numbers, err := stor.GetOrdersForAccrual(ctx)
	if err != nil {
		return nil, err
	}
	logger.ServerLog.Debug(fmt.Sprintf("have %d orders to update in accrual", len(numbers)))
	// заказов для обновления данных в системе accrual нет
	if len(numbers) == 0 {
		return make([]Result, 0), nil
	}

	// устанавливаю количество запросов
	numJobs := len(numbers)
	// создаем буферизованный канал для принятия задач в воркер
	jobs := make(chan int64, numJobs)
	// создаем буферизованный канал для отправки результатов
	results := make(chan Result, numJobs)

	// Запускаю одновременно 10 воркеров
	// можно определить этот параметр через глобальную переменную
	for w := 1; w <= 10; w++ {
		go Sender(jobs, results, GetAccrualSystemAddress(), client)
	}

	for _, num := range numbers {
		jobs <- num
	}
	// закрываю канал
	close(jobs)

	dataFromAccrual := make([]Result, 0)

	for i := 0; i < numJobs; i++ {
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

func UpdateAccrualData(ctx context.Context, stor repositories.OrderManager, retry repositories.RetryHandler) {
	// создаю нового клиента для отправки запросов к accrual
	client := resty.New()

	for {
		// Проверка отмены контекста
		select {
		case <-ctx.Done():
			return
		default:
			// Контекст ещё не отменен, можно продолжить выполнение
		}

		// Устанавливаю ожидание, если был превышен лимит обращений к accrual
		waitPeriod, err := retry.GetRetryPeriod(ctx, "accrual")
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// данные о периоде, в котором нужно прекратить обращения к accrual отсутствуют
				// можно продолжить работу
				logger.ServerLog.Debug("no data of retry period", zap.String("function", "UpdateAccrualData"))
			} else {
				// ошибка сервиса
				logger.ServerLog.Error("internal error in GetRetryPeriod", zap.String("error", err.Error()))
			}

		} else {
			// есть данные о дате, до которой нельзя обращаться к accrual
			logger.ServerLog.Debug("wait until", zap.String("date", waitPeriod.Format(time.RFC3339)))
			waitUntil(waitPeriod)
		}

		results, err := Generator(ctx, client, stor)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// заказы для отравки в систему расчета баллов отсутствуют
				// обрабатывать нечего, завершаю итерацию
				logger.ServerLog.Debug("no data of orders", zap.String("function", "UpdateAccrualData"))
				// ожидаю заказы
				time.Sleep(waitOrders)
				continue
			} else {
				// ошибка сервиса
				logger.ServerLog.Error("internal error in GetRetryPeriod", zap.String("error", err.Error()))
				// прерываю итерацию
				continue
			}
		}
		if len(results) == 0 {
			// заказы для отравки в систему расчета баллов отсутствуют
			// обрабатывать нечего, завершаю итерацию
			logger.ServerLog.Debug("no data of orders", zap.String("function", "UpdateAccrualData"))
			// ожидаю заказы
			time.Sleep(waitOrders)
			continue
		}
		logger.ServerLog.Debug("have orders to request for accrual", zap.String("count", fmt.Sprintf("%d", len(results))))

		accrualData := make([]repositories.AccrualData, 0)

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
			if result.requestStatus == http.StatusOK {
				// накапливаю данные для обновления заказов в рамках единой транзакции
				accrualData = append(accrualData, repositories.AccrualData{Order: result.Order, Status: result.Status, Accrual: result.Accrual})
			}
		}
		// обновляю информацию в заказах в случае если есть данные для обновления
		if len(accrualData) > 0 {
			err = stor.UpdateOrderTX(ctx, accrualData)
			if err != nil {
				logger.ServerLog.Error("fail to update data in gophermart", zap.String("error", err.Error()))
			}
		}

		// ожидание между итерациями отправки запросов к accrual
		time.Sleep(requestPeriod)
	}
}
