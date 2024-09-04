package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

var secretKey = "defaultSecretKey"

func SetSecretKey(newKey string) {
	secretKey = newKey
}

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
