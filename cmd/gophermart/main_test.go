package main

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"os"
// 	"os/exec"
// 	"testing"
// 	"time"

// 	"github.com/AntonBezemskiy/gophermart/internal/repositories"
// 	"github.com/go-resty/resty/v2"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// type Good struct {
// 	Description string  `json:"description"`
// 	Price       float64 `json:"price"`
// }

// type Request struct {
// 	Order string `json:"order"`
// 	Goods []Good `json:"goods"`
// }

// type Reward struct {
// 	Match      string  `json:"match"`
// 	Reward     float64 `json:"reward"`
// 	RewardType string  `json:"reward_type"`
// }

// func TestGophermart(t *testing.T) {
// 	// Запускаю сервис accrual--------------------------------------------------
// 	accrualAdress := ":8081"
// 	cmdAccrual := exec.Command("./../accrual/accrual_linux_amd64", fmt.Sprintf("-a=%s", accrualAdress))
// 	// Связываем стандартный вывод и ошибки программы с выводом программы Go
// 	cmdAccrual.Stdout = log.Writer()
// 	cmdAccrual.Stderr = log.Writer()
// 	// Запуск программы
// 	err := cmdAccrual.Start()
// 	require.NoError(t, err)

// 	// Запускаю gophermart-----------------------------------------------------
// 	gophermartAdress := ":8080"
// 	cmdGophermart := exec.Command("./gophermart", fmt.Sprintf("-a=%s", gophermartAdress), fmt.Sprintf("-r=%s", accrualAdress))
// 	// Связываем стандартный вывод и ошибки программы с выводом программы Go
// 	cmdGophermart.Stdout = log.Writer()
// 	cmdGophermart.Stderr = log.Writer()
// 	// Запуск программы
// 	err = cmdGophermart.Start()
// 	require.NoError(t, err)

// 	time.Sleep(2 * time.Second) // Жду 2 секунды для запуска сервиса

// 	// тест приложения gophermart
// 	{
// 		// Запоминаю токен, который возвращает gophermart при успешной регистрации нового пользователя
// 		var tokenUser1 string

// 		// наполняю сервис accrual данными для начисления баллов
// 		{
// 			// создаю нового клиента
// 			client := resty.New()

// 			// регистрирую информацию о вознаграждении за заказ
// 			reward := Reward{
// 				Match:      "Asus",
// 				Reward:     5000,
// 				RewardType: "pt",
// 			}
// 			var r bytes.Buffer
// 			enc := json.NewEncoder(&r)
// 			err := enc.Encode(reward)
// 			require.NoError(t, err)
// 			// отправляю заказ в систему
// 			resp, err := client.R().
// 				SetHeader("Content-Type", "application/json").
// 				SetBody(r.Bytes()).
// 				Post("http://localhost:8081/api/goods")
// 			require.NoError(t, err)
// 			assert.Equal(t, 200, resp.StatusCode())
// 		}
// 		// проверка регистрации пользователя в системе
// 		{
// 			// регистрирую нового пользователя, код 200 ----
// 			{
// 				client := resty.New()
// 				reg1 := repositories.AuthData{
// 					Login:    "user1",
// 					Password: "password1",
// 				}
// 				var r bytes.Buffer
// 				enc := json.NewEncoder(&r)
// 				err := enc.Encode(reg1)
// 				require.NoError(t, err)
// 				// отправляю запрос в систему
// 				resp, err := client.R().
// 					SetHeader("Content-Type", "application/json").
// 					SetBody(r.Bytes()).
// 					Post(fmt.Sprintf("http://%s/api/user/register", gophermartAdress))
// 				require.NoError(t, err)
// 				assert.Equal(t, 200, resp.StatusCode())

// 				// получаю токен от сервера, который находится в куки
// 				cookies := resp.Cookies()
// 				for _, cookie := range cookies {
// 					if cookie.Name == "token" {
// 						tokenUser1 = cookie.Value
// 					}
// 				}
// 			}

// 			// пытаюсь зарегистрировать нового пользователя с логином, который уже занят
// 			{
// 				client := resty.New()
// 				reg2 := repositories.AuthData{
// 					Login:    "user1",
// 					Password: "password2",
// 				}
// 				var r bytes.Buffer
// 				enc := json.NewEncoder(&r)
// 				err := enc.Encode(reg2)
// 				require.NoError(t, err)

// 				// отправляю запрос в систему
// 				resp, err := client.R().
// 					SetHeader("Content-Type", "application/json").
// 					SetBody(r.Bytes()).
// 					Post(fmt.Sprintf("http://%s/api/user/register", gophermartAdress))
// 				require.NoError(t, err)
// 				assert.Equal(t, 409, resp.StatusCode())
// 			}

// 			// пытаюсь зарегистрировать пользователя с некорректным логином
// 			{
// 				client := resty.New()
// 				reg3 := repositories.AuthData{
// 					Login:    "",
// 					Password: "password3",
// 				}
// 				var r bytes.Buffer
// 				enc := json.NewEncoder(&r)
// 				err := enc.Encode(reg3)
// 				require.NoError(t, err)

// 				// отправляю запрос в систему
// 				resp, err := client.R().
// 					SetHeader("Content-Type", "application/json").
// 					SetBody(r.Bytes()).
// 					Post(fmt.Sprintf("http://%s/api/user/register", gophermartAdress))
// 				require.NoError(t, err)
// 				assert.Equal(t, 400, resp.StatusCode())
// 			}
// 			// пытаюсь зарегистрировать пользователя с некорректным паролем
// 			{
// 				client := resty.New()
// 				reg4 := repositories.AuthData{
// 					Login:    "user4",
// 					Password: "",
// 				}
// 				var r bytes.Buffer
// 				enc := json.NewEncoder(&r)
// 				err := enc.Encode(reg4)
// 				require.NoError(t, err)

// 				// отправляю запрос в систему
// 				resp, err := client.R().
// 					SetHeader("Content-Type", "application/json").
// 					SetBody(r.Bytes()).
// 					Post(fmt.Sprintf("http://%s/api/user/register", gophermartAdress))
// 				require.NoError(t, err)
// 				assert.Equal(t, 400, resp.StatusCode())
// 			}
// 		}
// 		// тесты с аутентификацией пользователя
// 		{
// 			// случай успешной аутентификации
// 			{
// 				client := resty.New()
// 				reg1 := repositories.AuthData{
// 					Login:    "user1",
// 					Password: "password1",
// 				}
// 				var r bytes.Buffer
// 				enc := json.NewEncoder(&r)
// 				err := enc.Encode(reg1)
// 				require.NoError(t, err)
// 				// отправляю запрос в систему
// 				resp, err := client.R().
// 					SetHeader("Content-Type", "application/json").
// 					SetBody(r.Bytes()).
// 					Post(fmt.Sprintf("http://%s/api/user/login", gophermartAdress))
// 				require.NoError(t, err)
// 				assert.Equal(t, 200, resp.StatusCode())

// 				// получаю токен от сервера, который находится в куки
// 				var tokenLoginUser1 string
// 				cookies := resp.Cookies()
// 				for _, cookie := range cookies {
// 					if cookie.Name == "token" {
// 						tokenLoginUser1 = cookie.Value
// 					}
// 				}
// 				// проверяю на равенство токен полученный при регистрации и аутентификации
// 				assert.Equal(t, tokenUser1, tokenLoginUser1)
// 			}
// 			// случай с неверным форматом запроса
// 			{
// 				client := resty.New()
// 				reg3 := repositories.AuthData{
// 					Login:    "",
// 					Password: "password3",
// 				}
// 				var r bytes.Buffer
// 				enc := json.NewEncoder(&r)
// 				err := enc.Encode(reg3)
// 				require.NoError(t, err)

// 				// отправляю запрос в систему
// 				resp, err := client.R().
// 					SetHeader("Content-Type", "application/json").
// 					SetBody(r.Bytes()).
// 					Post(fmt.Sprintf("http://%s/api/user/login", gophermartAdress))
// 				require.NoError(t, err)
// 				assert.Equal(t, 400, resp.StatusCode())
// 			}
// 		}

// 	}

// 	// Останавливаем сервис accrual
// 	err = cmdAccrual.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
// 	require.NoError(t, err)

// 	// Останавливаем сервис gophermart
// 	err = cmdGophermart.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
// 	require.NoError(t, err)

// 	time.Sleep(3 * time.Second)
// }
