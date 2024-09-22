package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/AntonBezemskiy/gophermart/internal/pg"
	"github.com/AntonBezemskiy/gophermart/internal/repositories"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Good struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type Request struct {
	Order string `json:"order"`
	Goods []Good `json:"goods"`
}

type Reward struct {
	Match      string  `json:"match"`
	Reward     float64 `json:"reward"`
	RewardType string  `json:"reward_type"`
}

func TestGophermart(t *testing.T) {
	// Функция для очистки данных в базе
	cleanBD := func(dsn string) {
		// очищаю данные в тестовой бд------------------------------------------------------
		// создаём соединение с СУБД PostgreSQL
		conn, err := sql.Open("pgx", dsn)
		require.NoError(t, err)
		defer conn.Close()

		// Проверка соединения с БД
		ctx := context.Background()
		err = conn.PingContext(ctx)
		require.NoError(t, err)

		// создаем экземпляр хранилища pg
		stor := pg.NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}

	// функция для получения свободного порта для запуска приложений
	getFreePort := func() (int, error) {
		// Слушаем на порту 0, чтобы операционная система выбрала свободный порт
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return 0, err
		}
		defer listener.Close()

		// Получаем назначенный системой порт
		port := listener.Addr().(*net.TCPAddr).Port
		return port, nil
	}

	// Запускаю сервис accrual--------------------------------------------------
	accrualPort, err := getFreePort()
	require.NoError(t, err)
	accrualAdress := fmt.Sprintf("localhost:%d", accrualPort)
	cmdAccrual := exec.Command("./../accrual/accrual_linux_amd64", fmt.Sprintf("-a=%s", accrualAdress))
	// Связываем стандартный вывод и ошибки программы с выводом программы Go
	cmdAccrual.Stdout = log.Writer()
	cmdAccrual.Stderr = log.Writer()
	// Запуск программы
	err = cmdAccrual.Start()
	require.NoError(t, err)

	// Определяю параметры для запуска gophermart
	gophermartPort, err := getFreePort()
	require.NoError(t, err)
	gophermartAdress := fmt.Sprintf(":%d", gophermartPort)
	databaseDsn := "host=localhost user=endtestgophermart password=newpassword dbname=endtestgophermart sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)

	// Запускаю gophermart-----------------------------------------------------
	cmdGophermart := exec.Command("./gophermart", fmt.Sprintf("-a=%s", gophermartAdress),
		fmt.Sprintf("-r=%s", fmt.Sprintf("http://%s", accrualAdress)), fmt.Sprintf("-d=%s", databaseDsn), "-l=info")
	// Связываем стандартный вывод и ошибки программы с выводом программы Go
	cmdGophermart.Stdout = log.Writer()
	cmdGophermart.Stderr = log.Writer()
	// Запуск программы
	err = cmdGophermart.Start()
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // Жду 2 секунды для запуска сервиса

	// Запоминаю токен, который возвращает gophermart при успешной регистрации нового пользователя
	var tokenUser1 string
	// номер заказа для которого будут начисляться баллы
	succesOrder := "707261873236"
	//
	withdrawNumber1 := "155047814506"
	withdrawNumber2 := "657064403758"

	// наполняю сервис accrual данными для начисления баллов
	{
		// создаю нового клиента
		client := resty.New()

		// регистрирую информацию о вознаграждении за заказ
		reward := Reward{
			Match:      "Asus",
			Reward:     5000,
			RewardType: "pt",
		}
		var r bytes.Buffer
		enc := json.NewEncoder(&r)
		err := enc.Encode(reward)
		require.NoError(t, err)
		// отправляю заказ в систему
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(r.Bytes()).
			Post(fmt.Sprintf("http://%s/api/goods", accrualAdress))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
	}
	// добавляю заказ в систему accrual
	{
		client := resty.New()
		// добавляю в систему тестовый заказ
		request1 := Request{
			Order: succesOrder,
			Goods: []Good{
				{
					Description: "Laptop Asus",
					Price:       100500,
				},
			},
		}
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		err = enc.Encode(request1)
		require.NoError(t, err)

		// отправляю заказ в систему
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(b.Bytes()).
			Post(fmt.Sprintf("http://%s/api/orders", accrualAdress))
		require.NoError(t, err)
		assert.Equal(t, 202, resp.StatusCode())
	}

	// тест приложения gophermart
	{

		// проверка регистрации пользователя в системе
		{
			// регистрирую нового пользователя, код 200 ----
			{
				client := resty.New()
				reg1 := repositories.AuthData{
					Login:    "user1",
					Password: "password1",
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(reg1)
				require.NoError(t, err)
				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					Post(fmt.Sprintf("http://%s/api/user/register", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// получаю токен от сервера, который находится в куки
				cookies := resp.Cookies()
				for _, cookie := range cookies {
					if cookie.Name == "token" {
						tokenUser1 = cookie.Value
					}
				}
			}

			// пытаюсь зарегистрировать нового пользователя с логином, который уже занят
			{
				client := resty.New()
				reg2 := repositories.AuthData{
					Login:    "user1",
					Password: "password2",
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(reg2)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					Post(fmt.Sprintf("http://%s/api/user/register", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 409, resp.StatusCode())
			}

			// пытаюсь зарегистрировать пользователя с некорректным логином
			{
				client := resty.New()
				reg3 := repositories.AuthData{
					Login:    "",
					Password: "password3",
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(reg3)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					Post(fmt.Sprintf("http://%s/api/user/register", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 400, resp.StatusCode())
			}
			// пытаюсь зарегистрировать пользователя с некорректным паролем
			{
				client := resty.New()
				reg4 := repositories.AuthData{
					Login:    "user4",
					Password: "",
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(reg4)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					Post(fmt.Sprintf("http://%s/api/user/register", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 400, resp.StatusCode())
			}
		}
		// тесты с аутентификацией пользователя
		{
			// случай успешной аутентификации
			{
				client := resty.New()
				reg1 := repositories.AuthData{
					Login:    "user1",
					Password: "password1",
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(reg1)
				require.NoError(t, err)
				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					Post(fmt.Sprintf("http://%s/api/user/login", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// получаю токен от сервера, который находится в куки
				var tokenLoginUser1 string
				cookies := resp.Cookies()
				for _, cookie := range cookies {
					if cookie.Name == "token" {
						tokenLoginUser1 = cookie.Value
					}
				}
				// проверяю на равенство токен полученный при регистрации и аутентификации
				assert.Equal(t, tokenUser1, tokenLoginUser1)
			}
			// случай с неверным форматом запроса
			{
				client := resty.New()
				reg3 := repositories.AuthData{
					Login:    "",
					Password: "password3",
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(reg3)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					Post(fmt.Sprintf("http://%s/api/user/login", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 400, resp.StatusCode())
			}
			// случай с неверным паролем
			{
				client := resty.New()
				reg2 := repositories.AuthData{
					Login:    "user1",
					Password: "password2",
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(reg2)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					Post(fmt.Sprintf("http://%s/api/user/login", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 401, resp.StatusCode())
			}
		}
		// тесты до загрузки заказов
		{
			// отправка запроса незарегистрированного пользователя
			{
				client := resty.New()
				// отправляю запрос в систему
				resp, err := client.R().
					Get(fmt.Sprintf("http://%s/api/user/orders", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 401, resp.StatusCode())
			}
			// отправка запроса незарегистрированного пользователя
			{
				client := resty.New()
				// отправляю запрос в систему
				resp, err := client.R().
					Get(fmt.Sprintf("http://%s/api/user/balance", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 401, resp.StatusCode())
			}
			// попытка получить информацию о выводе средств незарегистрированным пользователем
			{
				client := resty.New()

				// отправляю запрос в систему
				resp, err := client.R().
					Get(fmt.Sprintf("http://%s/api/user/withdrawals", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 401, resp.StatusCode())
			}
			// получение списка загруженных заказов зарегистрированного пользователя
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/orders", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 204, resp.StatusCode())
			}
			// получение баланса зарегистрированным пользователем
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/balance", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// получение баланса
				var balance repositories.Balance
				b := bytes.NewBuffer(resp.Body())
				dec := json.NewDecoder(b)
				err = dec.Decode(&balance)
				require.NoError(t, err)

				// проверка баланса
				assert.Equal(t, 0.0, balance.Current)
				assert.Equal(t, 0.0, balance.Withdrawn)
			}
			// Попытка вывода средств с неверным номером заказа
			{

				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				w := repositories.WithdrawRequest{
					Order: "wrong number",
					Sum:   400,
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(w)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					SetCookie(cookie).
					Post(fmt.Sprintf("http://%s/api/user/balance/withdraw", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 422, resp.StatusCode())
			}
			// получение информации о выводе средств зарегистрированным пользователем
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/withdrawals", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 204, resp.StatusCode())
			}
		}
		// тесты с загрузкой заказов
		{
			// успешная загрузка с начислением баллов
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "text/plain").
					SetBody(succesOrder).
					SetCookie(cookie).
					Post(fmt.Sprintf("http://%s/api/user/orders", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 202, resp.StatusCode())
			}
			// попытка повторной загрузки заказа тем-же пользователем
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "text/plain").
					SetBody(succesOrder).
					SetCookie(cookie).
					Post(fmt.Sprintf("http://%s/api/user/orders", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())
			}
			// загрузка заказа: неверный формат запроса
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "text/plain").
					SetCookie(cookie).
					Post(fmt.Sprintf("http://%s/api/user/orders", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 422, resp.StatusCode())
			}
			// загрузка заказа: неверный формат номера заказа
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "text/plain").
					SetBody("123").
					SetCookie(cookie).
					Post(fmt.Sprintf("http://%s/api/user/orders", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 422, resp.StatusCode())
			}
		}
		// тесты после загрузки заказов
		{
			// Ожидание, чтобы gophermart успел обновить информацию в заказах, полученную от accrual
			time.Sleep(3 * time.Second)

			// получение списка загруженных заказов зарегистрированного пользователя
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/orders", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// список заказов полученный от сервиса gophermart
				ordersGet := make([]repositories.Order, 0)
				// желаемый список заказов для проверки
				ordersWant := []repositories.Order{
					{
						Number:  succesOrder,
						Status:  repositories.PROCESSED,
						Accrual: 5000,
					},
				}

				b := bytes.NewBuffer(resp.Body())
				dec := json.NewDecoder(b)
				err = dec.Decode(&ordersGet)
				require.NoError(t, err)

				// проверка списка заказов
				assert.Equal(t, ordersWant[0].Number, ordersGet[0].Number)
				assert.Equal(t, ordersWant[0].Status, ordersGet[0].Status)
				assert.InEpsilon(t, ordersWant[0].Accrual, ordersGet[0].Accrual, 0.0001)
			}
			// получение баланса зарегистрированным пользователем
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/balance", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// получение баланса
				var balance repositories.Balance
				b := bytes.NewBuffer(resp.Body())
				dec := json.NewDecoder(b)
				err = dec.Decode(&balance)
				require.NoError(t, err)

				// проверка баланса
				assert.InEpsilon(t, 5000.0, balance.Current, 0.0001)
				assert.Equal(t, 0.0, balance.Withdrawn)
			}
			// получение информации о выводе средств зарегистрированным пользователем
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/withdrawals", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 204, resp.StatusCode())
			}
			// Попытка вывода средств превыщающих текущий баланс
			{

				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				w := repositories.WithdrawRequest{
					Order: "155047814506",
					Sum:   50001,
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(w)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					SetCookie(cookie).
					Post(fmt.Sprintf("http://%s/api/user/balance/withdraw", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 402, resp.StatusCode())
			}
			// УСпешный вывод средств
			{

				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				w := repositories.WithdrawRequest{
					Order: withdrawNumber1,
					Sum:   3400,
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(w)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					SetCookie(cookie).
					Post(fmt.Sprintf("http://%s/api/user/balance/withdraw", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())
			}
			// получение баланса зарегистрированным пользователем после списания средств
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/balance", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// получение баланса
				var balance repositories.Balance
				b := bytes.NewBuffer(resp.Body())
				dec := json.NewDecoder(b)
				err = dec.Decode(&balance)
				require.NoError(t, err)

				// проверка баланса
				assert.InEpsilon(t, 1600.0, balance.Current, 0.0001)
				assert.InEpsilon(t, 3400.0, balance.Withdrawn, 0.0001)
			}
			// получение информации о выводе средств зарегистрированным пользователем
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/withdrawals", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// Желаемый информация о выводе средств
				w := repositories.WithdrawRequest{
					Order: withdrawNumber1,
					Sum:   3400,
				}
				// информация о выводе средств, полученная от gophermart
				wGet := make([]repositories.Withdrawals, 0)
				b := bytes.NewBuffer(resp.Body())
				dec := json.NewDecoder(b)
				err = dec.Decode(&wGet)
				require.NoError(t, err)

				assert.Equal(t, 1, len(wGet))
				assert.Equal(t, w.Order, wGet[0].Order)
				assert.InEpsilon(t, w.Sum, wGet[0].Sum, 0.0001)
			}
			// Ещё один успешный вывод средств
			{

				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				w := repositories.WithdrawRequest{
					Order: withdrawNumber2,
					Sum:   700,
				}
				var r bytes.Buffer
				enc := json.NewEncoder(&r)
				err := enc.Encode(w)
				require.NoError(t, err)

				// отправляю запрос в систему
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					SetBody(r.Bytes()).
					SetCookie(cookie).
					Post(fmt.Sprintf("http://%s/api/user/balance/withdraw", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())
			}
			// получение баланса зарегистрированным пользователем после второго списания средств
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/balance", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// получение баланса
				var balance repositories.Balance
				b := bytes.NewBuffer(resp.Body())
				dec := json.NewDecoder(b)
				err = dec.Decode(&balance)
				require.NoError(t, err)

				// проверка баланса
				assert.InEpsilon(t, 900.0, balance.Current, 0.0001)
				assert.InEpsilon(t, 4100.0, balance.Withdrawn, 0.0001)
			}
			// получение информации о выводе средств зарегистрированным пользователем
			{
				client := resty.New()
				// Создаю объект куки
				cookie := &http.Cookie{
					Name:  "token",
					Value: tokenUser1,
				}

				// отправляю запрос в систему
				resp, err := client.R().
					SetCookie(cookie).
					Get(fmt.Sprintf("http://%s/api/user/withdrawals", gophermartAdress))
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode())

				// Желаемый информация о выводе средств
				w := []repositories.WithdrawRequest{
					{
						Order: withdrawNumber2,
						Sum:   700,
					},
					{
						Order: withdrawNumber1,
						Sum:   3400,
					},
				}
				// информация о выводе средств, полученная от gophermart
				wGet := make([]repositories.Withdrawals, 0)
				b := bytes.NewBuffer(resp.Body())
				dec := json.NewDecoder(b)
				err = dec.Decode(&wGet)
				require.NoError(t, err)

				assert.Equal(t, 2, len(wGet))
				assert.Equal(t, w[0].Order, wGet[0].Order)
				assert.InEpsilon(t, w[0].Sum, wGet[0].Sum, 0.0001)
				assert.Equal(t, w[1].Order, wGet[1].Order)
				assert.InEpsilon(t, w[1].Sum, wGet[1].Sum, 0.0001)

			}
		}

	}

	// Останавливаем сервис accrual
	err = cmdAccrual.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
	require.NoError(t, err)

	// Останавливаем сервис gophermart
	err = cmdGophermart.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
	require.NoError(t, err)
	time.Sleep(3 * time.Second)

	// Очищаю бд от тестовых данных
	cleanBD(databaseDsn)

}
