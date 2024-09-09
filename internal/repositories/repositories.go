package repositories

import "context"

type AuthInterface interface {
	Register(context.Context, string, string) (bool, string, error)     // return values is Ok, token and error. Ok is false when login of user is not unique
	Authenticate(context.Context, string, string) (bool, string, error) // return values is Ok, token and error. Ok is false when login or password of user is wrong
}

type AuthData struct {
	Login    string `json:"loggin"`   // логин пользователя
	Password string `json:"password"` // пароль пользователя
}

func NewAuthData(login, password string) *AuthData {
	return &AuthData{
		Login:    login,
		Password: password,
	}
}

// -------------------------------------------------------------------------------------------------

const (
	ORDERSCODE200 = 200 // order already has been loaded of this user
	ORDERSCODE202 = 202 // new order has been got in processing
	ORDERSCODE409 = 409 // order already has been loaded of other user
	ORDERSCODE422 = 422 // wrong type of number of order
)

type OredersInterface interface {
	Load(context.Context, string) (int, error) // load order in storage. Return different status codes and posible error
}
