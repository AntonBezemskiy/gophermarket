package repositories

import "context"

type AuthInterface interface {
	Register(context.Context, string, string) (bool, string, error) // return values is Ok, token and error. Ok is false when login of user is not unique
	Authenticate(context.Context, string, string) (bool, string, error) // return values is Ok, token and error. Ok is false when login or password of user is wrong
}

type AuthData struct {
	Login    string `json:"loggin"`   // логин пользователя
	Password string `json:"password"` // пароль пользователя
}

func NewAuthData(login, password string) *AuthData{
	return &AuthData{
		Login: login,
		Password: password,
	}
}
