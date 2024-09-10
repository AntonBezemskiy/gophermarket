package repositories

import (
	"context"
	"time"
)

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

const (
	GETORDERSCODE200 = 200 // successful getting of orders
	GETORDERSCODE204 = 204 // data for answering not exist
)

type Order struct {
	Number     int64     `json:"number"`      // номер заказа
	Status     string    `json:"status"`      // стату обработки заказа
	Accrual    int       `json:"accrual"`     // сумма начисленных бонусов за заказ
	UploadedAt time.Time `json:"uploaded_at"` // время загрузки заказа
}

type OredersInterface interface {
	Load(context.Context, string, string) (int, error) // load order in storage. Gets context, id of user, order number of user. Returns different status codes and posible error
	Get(context.Context, string) ([]Order, int, error) // get list of orders. List is sortied by data of loading. Gets context, id of user. Returns list, status and error
}
