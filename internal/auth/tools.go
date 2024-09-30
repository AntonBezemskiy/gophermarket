package auth

import (
	"fmt"
	"net/http"
	"time"
)

func SetTokenCookie(w http.ResponseWriter, token string) {
	cookie := http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour), // срок действия cookie
		HttpOnly: true,                           // токен будет недоступен через JavaScript
		Path:     "/",                            // путь, по которому доступен cookie
	}

	http.SetCookie(w, &cookie)
}

func GetTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// GetTokenFromResponseCookie извлекает JWT-токен из куки в ответе сервера
// необходима для тестирования хэнлеров сервера. Имитирую работу клиента и получение им токена из куки
func GetTokenFromResponseCookie(resp *http.Response) (string, error) {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "token" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("token cookie not found")
}
