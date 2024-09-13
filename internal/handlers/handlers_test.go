package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AntonBezemskiy/gophermart/internal/auth"
	"github.com/AntonBezemskiy/gophermart/internal/mocks"
	"github.com/AntonBezemskiy/gophermart/internal/repositories"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotFound(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name    string
		request string
		want    want
	}{
		{
			name:    "Global addres",
			request: "/",
			want: want{
				code:        404,
				contentType: "text/plain",
			},
		},
		{
			name:    "Whrong addres",
			request: "/whrong",
			want: want{
				code:        404,
				contentType: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			w := httptest.NewRecorder()
			NotFound(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
		})
	}
}

func TestRegister(t *testing.T) {
	// Тест с моками
	{
		// Тест успешной авторизации, код 200

		// создаём контроллер
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockAuthInterface(ctrl)

		// тестовый случай с кодом 200---------------------------------------------------------
		// Создаю тело запроса с логином и паролем пользователя
		authData := *repositories.NewAuthData("successLogin", "successPassword")
		// сериализую струтктуру с логином и паролем в json-представление в виде слайса байт
		var bufEncode200 bytes.Buffer
		enc := json.NewEncoder(&bufEncode200)
		err := enc.Encode(authData)
		require.NoError(t, err)

		token, err := auth.BuildJWTString(24)
		require.NoError(t, err)
		m.EXPECT().Register(gomock.Any(), "successLogin", "successPassword").Return(true, token, nil)

		// тестовый случай с кодом 409-------------------------------------------------------------------
		m.EXPECT().Register(gomock.Any(), "loginIsUsed", "password").Return(false, "", nil)
		// Создаю тело запроса с логином и паролем пользователя
		authData = *repositories.NewAuthData("loginIsUsed", "password")
		// сериализую струтктуру с логином и паролем в json-представление в виде слайса байт
		var bufEncode409 bytes.Buffer
		enc = json.NewEncoder(&bufEncode409)
		err = enc.Encode(authData)
		require.NoError(t, err)

		// тестовый случай с кодом 500-------------------------------------------------------------------
		m.EXPECT().Register(gomock.Any(), "loginError", "password").Return(true, "", fmt.Errorf("test case of 500 status"))
		// Создаю тело запроса с логином и паролем пользователя
		authData = *repositories.NewAuthData("loginError", "password")
		// сериализую струтктуру с логином и паролем в json-представление в виде слайса байт
		var bufEncode500 bytes.Buffer
		enc = json.NewEncoder(&bufEncode500)
		err = enc.Encode(authData)
		require.NoError(t, err)

		tests := []struct {
			name       string
			login      string
			password   string
			body       io.Reader
			wantStatus int
			wantError  bool
			wantToken  string
		}{
			{
				name:       "succes register, status 200",
				body:       &bufEncode200,
				wantStatus: 200,
				wantError:  false,
				wantToken:  token,
			},
			{
				name:       "wrong format of request, status 400",
				body:       nil,
				wantStatus: 400,
				wantError:  true,
			},
			{
				name:       "login already is used, status 409",
				body:       &bufEncode409,
				wantStatus: 409,
				wantError:  true,
			},
			{
				name:       "internal server error, status 500",
				body:       &bufEncode500,
				wantStatus: 500,
				wantError:  true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Post("/api/user/register", RegisterHandler(m))

				request := httptest.NewRequest(http.MethodPost, "/api/user/register", tt.body)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, request)

				res := w.Result()
				defer res.Body.Close() // Закрываем тело ответа

				// Проверяю код ответа
				assert.Equal(t, tt.wantStatus, res.StatusCode)

				if tt.wantStatus == 200 {
					// проверяю токен, отправленный сервером
					getToken, err := auth.GetTokenFromResponseCookie(res)
					require.NoError(t, err)
					assert.Equal(t, tt.wantToken, getToken)
				}
				if tt.wantError {
					assert.NotEqual(t, 200, res.StatusCode)
				}
			})
		}
	}
}

func TestAuthentication(t *testing.T) {
	// Тест с моками
	{
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockAuthInterface(ctrl)

		// тестовый случай с кодом 200---------------------------------------------------------
		// Создаю тело запроса с логином и паролем пользователя
		authData := *repositories.NewAuthData("successLogin", "successPassword")
		// сериализую струтктуру с логином и паролем в json-представление в виде слайса байт
		var bufEncode200 bytes.Buffer
		enc := json.NewEncoder(&bufEncode200)
		err := enc.Encode(authData)
		require.NoError(t, err)

		token, err := auth.BuildJWTString(24)
		require.NoError(t, err)
		m.EXPECT().Authenticate(gomock.Any(), "successLogin", "successPassword").Return(true, token, nil)

		// тестовый случай с кодом 401-------------------------------------------------------------------
		m.EXPECT().Authenticate(gomock.Any(), "loginIsUsed", "password").Return(false, "", nil)
		// Создаю тело запроса с логином и паролем пользователя
		authData = *repositories.NewAuthData("loginIsUsed", "password")
		// сериализую струтктуру с логином и паролем в json-представление в виде слайса байт
		var bufEncode401 bytes.Buffer
		enc = json.NewEncoder(&bufEncode401)
		err = enc.Encode(authData)
		require.NoError(t, err)

		// тестовый случай с кодом 500-------------------------------------------------------------------
		m.EXPECT().Authenticate(gomock.Any(), "loginError", "password").Return(true, "", fmt.Errorf("test case of 500 status"))
		// Создаю тело запроса с логином и паролем пользователя
		authData = *repositories.NewAuthData("loginError", "password")
		// сериализую струтктуру с логином и паролем в json-представление в виде слайса байт
		var bufEncode500 bytes.Buffer
		enc = json.NewEncoder(&bufEncode500)
		err = enc.Encode(authData)
		require.NoError(t, err)

		tests := []struct {
			name       string
			login      string
			password   string
			body       io.Reader
			wantStatus int
			wantError  bool
			wantToken  string
		}{
			{
				name:       "succes authentication, status 200",
				body:       &bufEncode200,
				wantStatus: 200,
				wantError:  false,
				wantToken:  token,
			},
			{
				name:       "wrong format of request, status 400",
				body:       nil,
				wantStatus: 400,
				wantError:  true,
			},
			{
				name:       "login or password is wrong, status 409",
				body:       &bufEncode401,
				wantStatus: 401,
				wantError:  true,
			},
			{
				name:       "internal server error, status 500",
				body:       &bufEncode500,
				wantStatus: 500,
				wantError:  true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Post("/api/user/login", AuthenticationHandler(m))

				request := httptest.NewRequest(http.MethodPost, "/api/user/login", tt.body)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, request)

				res := w.Result()
				defer res.Body.Close() // Закрываем тело ответа

				// Проверяю код ответа
				assert.Equal(t, tt.wantStatus, res.StatusCode)

				if tt.wantStatus == 200 {
					// проверяю токен, отправленный сервером
					getToken, err := auth.GetTokenFromResponseCookie(res)
					require.NoError(t, err)
					assert.Equal(t, tt.wantToken, getToken)
				}
				if tt.wantError {
					assert.NotEqual(t, 200, res.StatusCode)
				}
			})
		}
	}
}

func TestLoadOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockOrdersInterface(ctrl)

	// тестовый случай с кодом 200--------------------------------------------------------
	m.EXPECT().Load(gomock.Any(), gomock.Any(), "succes order number, code 200").Return(200, nil)
	// Создаю тело запроса номером заказа
	orderNumb200 := []byte("succes order number, code 200")

	// тестовый случай с кодом 200--------------------------------------------------------
	m.EXPECT().Load(gomock.Any(), "success ID", "succes order number, code 200").Return(200, nil)
	// Создаю тело запроса номером заказа
	orderNumb200WithID := []byte("succes order number, code 200")

	// тестовый случай с кодом 202--------------------------------------------------------
	m.EXPECT().Load(gomock.Any(), gomock.Any(), "succes order number, code 202").Return(202, nil)
	// Создаю тело запроса номером заказа
	orderNumb202 := []byte("succes order number, code 202")

	// тестовый случай с кодом 409--------------------------------------------------------
	m.EXPECT().Load(gomock.Any(), gomock.Any(), "code 409").Return(409, nil)
	// Создаю тело запроса номером заказа
	orderNumb409 := []byte("code 409")

	// тестовый случай с кодом 409--------------------------------------------------------
	m.EXPECT().Load(gomock.Any(), gomock.Any(), "code 500").Return(0, fmt.Errorf("load order error"))
	// Создаю тело запроса номером заказа
	orderNumb500 := []byte("code 500")

	tests := []struct {
		name       string
		body       io.Reader
		id         string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "code 200",
			body:       bytes.NewReader(orderNumb200),
			id:         "test id",
			wantStatus: 200,
		},
		{
			name:       "code 200 with ID",
			body:       bytes.NewReader(orderNumb200WithID),
			id:         "success ID",
			wantStatus: 200,
		},
		{
			name:       "code 202",
			body:       bytes.NewReader(orderNumb202),
			wantStatus: 202,
		},
		{
			name:       "code 409",
			body:       bytes.NewReader(orderNumb409),
			wantStatus: 409,
		},
		{
			name:       "code 500",
			body:       bytes.NewReader(orderNumb500),
			wantStatus: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/api/user/orders", LoadOrdersHandler(m))

			request := httptest.NewRequest(http.MethodPost, "/api/user/orders", tt.body)
			w := httptest.NewRecorder()

			// Устанавливаю id пользователя в контекст
			ctx := context.WithValue(request.Context(), auth.UserIDKey, tt.id)

			r.ServeHTTP(w, request.WithContext(ctx))

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа

			// Проверяю код ответа
			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestGetOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockOrdersInterface(ctrl)

	// тестовый случай с кодом 200--------------------------------------------------------
	loadT := time.Date(2003, time.May, 1, 17, 1, 21, 0, time.UTC)

	order := repositories.Order{
		Number:     1234,
		Status:     "PROCESSED",
		Accrual:    500,
		UploadedAt: loadT,
	}
	orderSlice := []repositories.Order{order}
	m.EXPECT().Get(gomock.Any(), "id of 200 code").Return(orderSlice, repositories.GETORDERSCODE200, nil)

	// тестовый случай с кодом 204--------------------------------------------------------
	m.EXPECT().Get(gomock.Any(), "id of 204 code").Return(nil, repositories.GETORDERSCODE204, nil)

	// тестовый случай с кодом 500--------------------------------------------------------
	m.EXPECT().Get(gomock.Any(), "id of 500 code").Return(nil, 0, fmt.Errorf("error in get method"))

	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "code 200",
			id:         "id of 200 code",
			wantStatus: 200,
		},
		{
			name:       "code 204",
			id:         "id of 204 code",
			wantStatus: 204,
		},
		{
			name:       "code 500",
			id:         "id of 500 code",
			wantStatus: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/api/user/orders", GetOrdersHandler(m))

			request := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			w := httptest.NewRecorder()

			// Устанавливаю id пользователя в контекст
			ctx := context.WithValue(request.Context(), auth.UserIDKey, tt.id)

			r.ServeHTTP(w, request.WithContext(ctx))

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа

			// Проверяю код ответа
			assert.Equal(t, tt.wantStatus, res.StatusCode)

			// проверяю тело ответа в случае успешного кода запроса
			if tt.wantStatus == 200 {
				var resJSON = make([]repositories.Order, 0)
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				buRes := bytes.NewBuffer(body)
				dec := json.NewDecoder(buRes)
				err = dec.Decode(&resJSON)
				require.NoError(t, err)

				assert.Equal(t, orderSlice, resJSON)
			}
		})
	}
}

func TestGetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockBalanceInterface(ctrl)

	// тестовый случай с кодом 200--------------------------------------------------------
	balance := repositories.Balance{
		Current:   500,
		Withdrawn: 20.2,
	}
	m.EXPECT().Get(gomock.Any(), "id of 200 code").Return(balance, nil)

	// тестовый случай с кодом 500--------------------------------------------------------
	m.EXPECT().Get(gomock.Any(), "id of 500 code").Return(repositories.Balance{}, fmt.Errorf("error in get method"))

	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "code 200",
			id:         "id of 200 code",
			wantStatus: 200,
		},
		{
			name:       "code 500",
			id:         "id of 500 code",
			wantStatus: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/api/user/balance", GetBalanceHandler(m))

			request := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			w := httptest.NewRecorder()

			// Устанавливаю id пользователя в контекст
			ctx := context.WithValue(request.Context(), auth.UserIDKey, tt.id)

			r.ServeHTTP(w, request.WithContext(ctx))

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа

			// Проверяю код ответа
			assert.Equal(t, tt.wantStatus, res.StatusCode)

			// проверяю тело ответа в случае успешного кода запроса
			if tt.wantStatus == 200 {
				resJSON := repositories.Balance{}
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				buRes := bytes.NewBuffer(body)
				dec := json.NewDecoder(buRes)
				err = dec.Decode(&resJSON)
				require.NoError(t, err)

				assert.Equal(t, balance, resJSON)
			}
		})
	}
}

func TestWithdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockWithdrawInterface(ctrl)

	// тестовый случай с кодом 200--------------------------------------------------------
	withdraw200 := repositories.WithdrawRequest{
		Order: "1234",
		Sum:   100,
	}
	var body200 bytes.Buffer
	enc := json.NewEncoder(&body200)
	err := enc.Encode(withdraw200)
	require.NoError(t, err)

	m.EXPECT().Withdraw(gomock.Any(), "id of 200 code", "1234", 100).Return(repositories.WITHDRAWCODE200, nil)

	// тестовый случай с кодом 402--------------------------------------------------------
	withdraw402 := repositories.WithdrawRequest{
		Order: "1234",
		Sum:   2000,
	}
	var body402 bytes.Buffer
	enc402 := json.NewEncoder(&body402)
	err = enc402.Encode(withdraw402)
	require.NoError(t, err)

	m.EXPECT().Withdraw(gomock.Any(), "id of 402 code", "1234", 2000).Return(repositories.WITHDRAWCODE402, nil)

	// тестовый случай с кодом 422--------------------------------------------------------
	withdraw422 := repositories.WithdrawRequest{
		Order: "wrrong number of order",
		Sum:   2000,
	}
	var body422 bytes.Buffer
	enc422 := json.NewEncoder(&body422)
	err = enc422.Encode(withdraw422)
	require.NoError(t, err)

	m.EXPECT().Withdraw(gomock.Any(), "id of 422 code", "wrrong number of order", 2000).Return(repositories.WITHDRAWCODE422, nil)

	// тестовый случай с кодом 500--------------------------------------------------------
	withdraw500 := repositories.WithdrawRequest{
		Order: "12345",
		Sum:   2000,
	}
	var body500 bytes.Buffer
	enc500 := json.NewEncoder(&body500)
	err = enc500.Encode(withdraw500)
	require.NoError(t, err)

	m.EXPECT().Withdraw(gomock.Any(), "id of 500 code", "12345", 2000).Return(0, fmt.Errorf("error"))

	tests := []struct {
		name       string
		body       io.Reader
		id         string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "code 200",
			body:       bytes.NewReader(body200.Bytes()),
			id:         "id of 200 code",
			wantStatus: 200,
		},
		{
			name:       "code 402",
			body:       bytes.NewReader(body402.Bytes()),
			id:         "id of 402 code",
			wantStatus: 402,
		},
		{
			name:       "code 422",
			body:       bytes.NewReader(body422.Bytes()),
			id:         "id of 422 code",
			wantStatus: 422,
		},
		{
			name:       "code 500",
			body:       bytes.NewReader(body500.Bytes()),
			id:         "id of 500 code",
			wantStatus: 500,
		},
		{
			name:       "code 400",
			body:       bytes.NewReader([]byte("wrong body")),
			id:         "id of 400 code",
			wantStatus: 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/api/user/balance/withdraw", WithdrawHandler(m))

			request := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", tt.body)
			w := httptest.NewRecorder()

			// Устанавливаю id пользователя в контекст
			ctx := context.WithValue(request.Context(), auth.UserIDKey, tt.id)

			r.ServeHTTP(w, request.WithContext(ctx))

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа

			// Проверяю код ответа
			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockWithdrawalsInterface(ctrl)

	// тестовый случай с кодом 200--------------------------------------------------------
	loadT := time.Date(2003, time.May, 1, 17, 1, 21, 0, time.UTC)
	withdrawals := []repositories.Withdrawals{
		{
			Order:     "1234",
			Sum:       100,
			ProcessAt: loadT,
		},
	}

	m.EXPECT().Get(gomock.Any(), "id of 200 code").Return(withdrawals, repositories.WITHDRAWALS200, nil)

	// тестовый случай с кодом 204--------------------------------------------------------
	m.EXPECT().Get(gomock.Any(), "id of 204 code").Return(nil, repositories.WITHDRAWALS204, nil)

	// тестовый случай с кодом 500--------------------------------------------------------
	m.EXPECT().Get(gomock.Any(), "id of 500 code").Return(nil, 0, fmt.Errorf("error"))

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "code 200",
			id:         "id of 200 code",
			wantStatus: 200,
		},
		{
			name:       "code 204",
			id:         "id of 204 code",
			wantStatus: 204,
		},
		{
			name:       "code 500",
			id:         "id of 500 code",
			wantStatus: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/api/user/withdrawals", WithdrawalsHandler(m))

			request := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
			w := httptest.NewRecorder()

			// Устанавливаю id пользователя в контекст
			ctx := context.WithValue(request.Context(), auth.UserIDKey, tt.id)

			r.ServeHTTP(w, request.WithContext(ctx))

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа

			// Проверяю код ответа
			assert.Equal(t, tt.wantStatus, res.StatusCode)

			// проверяю тело ответа в случае успешного кода запроса
			if tt.wantStatus == 200 {
				resJSON := make([]repositories.Withdrawals, 0)
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				buRes := bytes.NewBuffer(body)
				dec := json.NewDecoder(buRes)
				err = dec.Decode(&resJSON)
				require.NoError(t, err)

				assert.Equal(t, withdrawals, resJSON)
			}
		})
	}
}
