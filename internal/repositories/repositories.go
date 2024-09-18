package repositories

import (
	"context"
	"time"
)

// Статусы обработки заказа в виде строки, чтобы информация в базе даных выглядела более информативно
const (
	NEW        = "NEW"
	PROCESSING = "PROCESSING"
	INVALID    = "INVALID"
	PROCESSED  = "PROCESSED"
)

type AuthInterface interface {
	Register(ctx context.Context, login string, password string) (ok bool, token string, err error)     // return values is Ok, token and error. Ok is false when login of user is not unique
	Authenticate(ctx context.Context, login string, password string) (ok bool, token string, err error) // return values is Ok, token and error. Ok is false when login or password of user is wrong
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
	Accrual    float64   `json:"accrual"`     // сумма начисленных бонусов за заказ
	UploadedAt time.Time `json:"uploaded_at"` // время загрузки заказа
}

type OrdersInterface interface {
	Load(ctx context.Context, idUser string, orderNumber string) (status int, err error)  // load order in storage. Gets context, id of user, order number of user. Returns different status codes and posible error
	GetOrders(ctx context.Context, idUser string) (orders []Order, status int, err error) // get list of orders. List is sortied by data of loading. Gets context, id of user. Returns list, status and error
	GetOrdersForAccrual(ctx context.Context) (numbers []int64, err error)                 // return numbers of all orders with NEW and PROCESSING status
}

//--------------------------------------------------------------------------------------------------------------------

type Balance struct {
	Current   float64 `json:"current"`   // текущая сумма баллов лояльности
	Withdrawn float64 `json:"withdrawn"` // сумма использованных баллов за весь период регистрации
}

type BalanceInterface interface {
	GetBalance(ctx context.Context, idUser string) (balance Balance, err error) // For getting current balance of user. Gets context, id of user. Returns balance and error.
}

//-----------------------------------------------------------------------------------------------------

const (
	WITHDRAWCODE200 = 200 // успешная обработка запроса
	WITHDRAWCODE402 = 402 // на счету недостаточно средств
	WITHDRAWCODE422 = 422 // неверный номер заказа
)

type WithdrawRequest struct {
	Order string  `json:"order"` // номер заказа
	Sum   float64 `json:"sum"`   // сумма баллов к списанию в счёт оплаты
}

type WithdrawInterface interface {
	Withdraw(ctx context.Context, idUser string, orderNumber string, sum float64) (status int, err error) // request for withdrawal of funds. Gets context, id of user, order number of user, sum to withdraw. Return code and error.
}

// -----------------------------------------------------------------------------------------------------
const (
	WITHDRAWALS200 = 200 // успешная обработка запроса
	WITHDRAWALS204 = 204 // нет ни одного списания
)

type Withdrawals struct {
	Order     string    `json:"order"`        // номер заказа
	Sum       float64   `json:"sum"`          // вывод средств
	ProcessAt time.Time `json:"processed_at"` // дата вывода средств
}

type WithdrawalsInterface interface {
	GetWithdrawals(ctx context.Context, idUser string) (withdrawals []Withdrawals, status int, err error) // get information about withdrawals. Gets context, id of user. Return list of withdrawals, status and error
}

// ------------------------------------------------------------------------------------------------------

type RetryInterface interface {
	AddRetryPeriod(ctx context.Context, service string, period time.Time) error // set period time of waiting for requesting to particular service
	GetRetryPeriod(ctx context.Context, service string) (time.Time, error)      // get period time of waiting for requesting to particular service
}
