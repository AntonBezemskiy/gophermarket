package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthChecker(t *testing.T) {
	// Случай успешной авторизации
	{
		userID := "success user ID"
		// Создаю отдельную реализацию билдера JWT, чтобы установить в него конкретный идентификатор
		buildJWT := func(expireHour int) string {
			// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					// когда создан токен
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expireHour))),
				},
				// собственное утверждение - идентификатор пользователя
				UserID: userID,
			})

			// создаём строку токена
			tokenString, err := token.SignedString([]byte(secretKey))
			require.NoError(t, err)
			return tokenString
		}

		token := buildJWT(1)

		cookie := http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(1 * time.Hour), // срок действия cookie
			HttpOnly: true,                          // токен будет недоступен через JavaScript
			Path:     "/",                           // путь, по которому доступен cookie
		}

		testHandler := func() http.HandlerFunc {
			fn := func(res http.ResponseWriter, req *http.Request) {
				// Извлекаю ID пользователя из контекста
				id, ok := req.Context().Value(UserIDKey).(string)
				require.Equal(t, true, ok)
				// Проверяю, что ID пользователя полученный из JWT токена совпадает с ожидаемым ID
				assert.Equal(t, userID, id)

				res.WriteHeader(http.StatusOK)
			}
			return fn
		}
		r := chi.NewRouter()
		r.Post("/api/user/orders", Checker(testHandler()))

		request := httptest.NewRequest(http.MethodPost, "/api/user/orders", nil)
		request.AddCookie(&cookie)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		result := w.Result()
		defer result.Body.Close()
		assert.Equal(t, http.StatusOK, result.StatusCode)
	}
	// Случай неуспешной авторизации. Пользователь не отправил JWT в запросе
	{
		testHandler := func() http.HandlerFunc {
			fn := func(res http.ResponseWriter, req *http.Request) {
				res.WriteHeader(http.StatusOK)
			}
			return fn
		}
		r := chi.NewRouter()
		r.Post("/api/user/orders", Checker(testHandler()))

		request := httptest.NewRequest(http.MethodPost, "/api/user/orders", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		result := w.Result()
		defer result.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	}
	// Случай неуспешной авторизации. Пользователь отправил неправильный JWT. JWT подписанный ключом, который отличается от ключа сервера
	{

		userID := "user ID"
		// Создаю отдельную реализацию билдера JWT, чтобы установить в него конкретный идентификатор
		buildJWT := func(expireHour int) string {
			// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					// когда создан токен
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expireHour))),
				},
				// собственное утверждение - идентификатор пользователя
				UserID: userID,
			})

			// создаём строку токена
			tokenString, err := token.SignedString([]byte("wrong secret key"))
			require.NoError(t, err)
			return tokenString
		}

		token := buildJWT(1)

		cookie := http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(1 * time.Hour), // срок действия cookie
			HttpOnly: true,                          // токен будет недоступен через JavaScript
			Path:     "/",                           // путь, по которому доступен cookie
		}

		testHandler := func() http.HandlerFunc {
			fn := func(res http.ResponseWriter, req *http.Request) {
				res.WriteHeader(http.StatusOK)
			}
			return fn
		}
		r := chi.NewRouter()
		r.Post("/api/user/orders", Checker(testHandler()))

		request := httptest.NewRequest(http.MethodPost, "/api/user/orders", nil)
		request.AddCookie(&cookie)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		result := w.Result()
		defer result.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	}
}
