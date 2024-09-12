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

type OrdersInterface interface {
	Load(context.Context, string, string) (int, error) // load order in storage. Gets context, id of user, order number of user. Returns different status codes and posible error
	Get(context.Context, string) ([]Order, int, error) // get list of orders. List is sortied by data of loading. Gets context, id of user. Returns list, status and error
}

//--------------------------------------------------------------------------------------------------------------------

type Balance struct {
	Current   float64 `json:"current"`   // текущая сумма баллов лояльности
	Withdrawn float64 `json:"withdrawn"` // сумма использованных баллов за весь период регистрации
}

type BalanceInterface interface {
	Get(context.Context, string) (Balance, error) // For getting current balance of user. Gets context, id of user. Returns balance and error.
}

//-----------------------------------------------------------------------------------------------------

const (
	WITHDRAWCODE200 = 200 // успешная обработка запроса
	WITHDRAWCODE402 = 402 // на счету недостаточно средств
	WITHDRAWCODE422 = 422 // неверный номер заказа
)

type WithdrawRequest struct {
	Order string `json:"order"` // номер заказа
	Sum   int    `json:"sum"`   // сумма баллов к списанию в счёт оплаты
}

type WithdrawInterface interface {
	Withdraw(context.Context, string, string, int) (int, error) // request for withdrawal of funds. Gets context, id of user, order number of user, sum to withdraw. Return code and error.
}

// -----------------------------------------------------------------------------------------------------
const (
	WITHDRAWALS200 = 200 // успешная обработка запроса
	WITHDRAWALS204 = 204 // нет ни одного списания
)

type Withdrawals struct {
	Order     string    `json:"order"`        // номер заказа
	Sum       int       `json:"sum"`          // вывод средств
	ProcessAt time.Time `json:"processed_at"` // дата вывода средств
}

type WithdrawalsInterface interface {
	Get(context.Context, string) ([]Withdrawals, int, error) // get information about withdrawals. Gets context, id of user. Return list of withdrawals, status and error
}
