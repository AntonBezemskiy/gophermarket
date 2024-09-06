package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AntonBezemskiy/gophermart/internal/logger"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var secretKey = "defaultSecretKey"

func SetSecretKey(newKey string) {
	secretKey = newKey
}

type contextKey string

const userIDKey = contextKey("userID")

// В качестве id пользователя будет использоваться сгенерированный UUID (Universally Unique Identifier)
// uuid.New() создает новое рандомное UUID или вызывает панику, кажется это может быть проблемой
func getUserID() string {
	id := uuid.New()
	return id.String()
}

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(expireHour int) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expireHour))),
		},
		// собственное утверждение - идентификатор пользователя
		UserID: getUserID(),
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

// Получение Id пользователя из токена c проверкой заголовка алгоритма токена.
// Заголовок должен совпадать с тем, который сервер использует для подписи и проверки токенов.
func GetUserID(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secretKey), nil
		})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("token is not valid")
	}

	return claims.UserID, nil
}

// Мидлварь для проверки JWT входящих запросов к серверу
// позволит установить доступ к ресурсам только для авторизированных пользователей
// Из полученного токена извлекаю ID пользователя у станавливаю этот идентификатор в контекст
func AuthChecker(h http.Handler) http.HandlerFunc {
	authFn := func(w http.ResponseWriter, r *http.Request) {
		token, err := GetTokenFromCookie(r)
		// В случае ошибки получить токен возвращаю возвращаю статус 401 - пользователь не аутентифицирован
		if err != nil {
			logger.ServerLog.Error("error of getting token in AuthChecker middleware", zap.String("address", r.URL.String()), zap.String("error", err.Error()))
			http.Error(w, "error of getting token", http.StatusUnauthorized)
			return
		}
		id, err := GetUserID(token)
		// В случае ошибки получить id пользователя возвращаю возвращаю статус 401 - пользователь не аутентифицирован
		if err != nil {
			logger.ServerLog.Error("error of getting id of user in AuthChecker middleware", zap.String("address", r.URL.String()), zap.String("error", err.Error()))
			http.Error(w, "error of getting id of user", http.StatusUnauthorized)
			return
		}
		// В случае успешного получения id пользователя устанавливаю идентификатор в контекст Value для дальней обработки сервером
		ctx := context.WithValue(r.Context(), userIDKey, id)
		// Передаю запрос дальше с изменённым контекстом
		h.ServeHTTP(w, r.WithContext(ctx))
	}
	return authFn
}
