package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/AntonBezemskiy/gophermart/internal/auth"
	"github.com/AntonBezemskiy/gophermart/internal/logger"
	"github.com/AntonBezemskiy/gophermart/internal/repositories"
	"go.uber.org/zap"
)

func NotFound(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusNotFound)
}

// Регистрация пользователя
func Register(res http.ResponseWriter, req *http.Request, regist repositories.AuthInterface) {
	res.Header().Set("Content-Type", "text/plain")
	defer req.Body.Close()

	var authData repositories.AuthData
	if err := json.NewDecoder(req.Body).Decode(&authData); err != nil {
		logger.ServerLog.Error("decode body error in Register handler", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	ok, token, err := regist.Register(req.Context(), authData.Login, authData.Password)
	if err != nil {
		logger.ServerLog.Error("register new user error", zap.String("address", req.URL.String()), zap.String("error: ", error.Error(err)))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		logger.ServerLog.Error("register of new user is failed", zap.String("address", req.URL.String()), zap.String("error", "login is not unique"))
		http.Error(res, "login is not unique", http.StatusConflict)
		return
	}

	// Для передачи аутентификационных данных использую механизм cookie
	auth.SetTokenCookie(res, token)
	res.WriteHeader(200)
}

// Аутентификация пользователя
func Authentication(res http.ResponseWriter, req *http.Request, authRep repositories.AuthInterface) {
	res.Header().Set("Content-Type", "text/plain")
	defer req.Body.Close()

	var authData repositories.AuthData
	if err := json.NewDecoder(req.Body).Decode(&authData); err != nil {
		logger.ServerLog.Error("decode body error in Authentication handler", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	ok, token, err := authRep.Authenticate(req.Context(), authData.Login, authData.Password)
	if err != nil {
		logger.ServerLog.Error("authentication of user error", zap.String("address", req.URL.String()), zap.String("error: ", error.Error(err)))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		logger.ServerLog.Error("authentication of user is failed", zap.String("address", req.URL.String()), zap.String("error", "login or password is wrong"))
		http.Error(res, "login or password is wrong", http.StatusUnauthorized)
		return
	}

	// Для передачи аутентификационных данных использую механизм cookie
	auth.SetTokenCookie(res, token)
	res.WriteHeader(200)
}

// Загрузка заказа в сервис
func LoadOrders(res http.ResponseWriter, req *http.Request, order repositories.OrdersInterface) {
	// Получаю id пользователя из контекста
	id, ok := req.Context().Value(auth.UserIDKey).(string)
	if !ok {
		logger.ServerLog.Error("user ID not found", zap.String("address", req.URL.String()))
		http.Error(res, "user ID not found", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	defer req.Body.Close()

	strOrderNumd, err := io.ReadAll(req.Body)
	if err != nil {
		logger.ServerLog.Error("can't get order number in LoadOrders", zap.String("address", req.URL.String()), zap.String("error", err.Error()))
		http.Error(res, "can't get order number in LoadOrders", http.StatusBadRequest)
		return
	}

	code, err := order.Load(req.Context(), id, string(strOrderNumd))
	if err != nil {
		logger.ServerLog.Error("load order error", zap.String("address", req.URL.String()), zap.String("error", err.Error()))
		http.Error(res, "load order error", http.StatusInternalServerError)
		return
	}

	switch code {
	case repositories.ORDERSCODE200:
		res.WriteHeader(http.StatusOK)
		return
	case repositories.ORDERSCODE202:
		res.WriteHeader(http.StatusAccepted)
		return
	case repositories.ORDERSCODE409:
		res.WriteHeader(http.StatusConflict)
		return
	case repositories.ORDERSCODE422:
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	default:
		logger.ServerLog.Error("load order error", zap.String("address", req.URL.String()), zap.String("error", "undefined return code from storage"))
		http.Error(res, "load order error", http.StatusInternalServerError)
		return
	}
}

// Выгрузка списка заказов пользователя отсортированных по времени загрузки
func GetOrders(res http.ResponseWriter, req *http.Request, order repositories.OrdersInterface) {
	// Получаю id пользователя из контекста
	id, ok := req.Context().Value(auth.UserIDKey).(string)
	if !ok {
		logger.ServerLog.Error("user ID not found", zap.String("address", req.URL.String()))
		http.Error(res, "user ID not found", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	defer req.Body.Close()

	orders, code, err := order.GetOrders(req.Context(), id)
	if err != nil {
		logger.ServerLog.Error("get orders error", zap.String("address", req.URL.String()), zap.String("error", err.Error()))
		http.Error(res, "get orders error", http.StatusInternalServerError)
		return
	}

	switch code {
	case repositories.GETORDERSCODE200:
		// устанавливаю заголовок таким образом вместо WriteHeader(http.StatusOK), потому что
		// далее в методе Write в middleware необходимо установить заголовок Hash со значением хэша,
		// а после WriteHeader заголовки уже не устанавливаются
		res.Header().Set("Status-Code", "200")
		enc := json.NewEncoder(res)
		if err := enc.Encode(orders); err != nil {
			logger.ServerLog.Error("error encoding response", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
			return
		}
	case repositories.GETORDERSCODE204:
		res.WriteHeader(http.StatusNoContent)
		return
	default:
		logger.ServerLog.Error("get orders error", zap.String("address", req.URL.String()), zap.String("error", "undefined return code from storage"))
		http.Error(res, "get orders error", http.StatusInternalServerError)
		return
	}
}

// Получение баланса пользователя
func GetBalance(res http.ResponseWriter, req *http.Request, blnc repositories.BalanceInterface) {
	// Получаю id пользователя из контекста
	id, ok := req.Context().Value(auth.UserIDKey).(string)
	if !ok {
		logger.ServerLog.Error("user ID not found", zap.String("address", req.URL.String()))
		http.Error(res, "user ID not found", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	defer req.Body.Close()

	balance, err := blnc.GetBalance(req.Context(), id)
	if err != nil {
		logger.ServerLog.Error("get balance error", zap.String("address", req.URL.String()), zap.String("error", err.Error()))
		http.Error(res, "get balance error", http.StatusInternalServerError)
		return
	}
	// устанавливаю заголовок таким образом вместо WriteHeader(http.StatusOK), потому что
	// далее в методе Write в middleware необходимо установить заголовок Hash со значением хэша,
	// а после WriteHeader заголовки уже не устанавливаются
	res.Header().Set("Status-Code", "200")
	enc := json.NewEncoder(res)
	if err := enc.Encode(balance); err != nil {
		logger.ServerLog.Error("error encoding response", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Запрос на списание средств
func Withdraw(res http.ResponseWriter, req *http.Request, wthdrw repositories.WithdrawInterface) {
	// Получаю id пользователя из контекста
	id, ok := req.Context().Value(auth.UserIDKey).(string)
	if !ok {
		logger.ServerLog.Error("user ID not found", zap.String("address", req.URL.String()))
		http.Error(res, "user ID not found", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	defer req.Body.Close()

	var withdraw repositories.WithdrawRequest
	if err := json.NewDecoder(req.Body).Decode(&withdraw); err != nil {
		logger.ServerLog.Error("decode body error", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	code, err := wthdrw.Withdraw(req.Context(), id, withdraw.Order, withdraw.Sum)
	if err != nil {
		logger.ServerLog.Error("request of withdraw process error", zap.String("address", req.URL.String()), zap.String("error", err.Error()))
		http.Error(res, "request of withdraw process error", http.StatusInternalServerError)
		return
	}

	switch code {
	case repositories.WITHDRAWCODE200:
		res.WriteHeader(http.StatusOK)
		return
	case repositories.WITHDRAWCODE402:
		res.WriteHeader(http.StatusPaymentRequired)
		return
	case repositories.WITHDRAWCODE422:
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	default:
		logger.ServerLog.Error("request of withdraw process error", zap.String("address", req.URL.String()), zap.String("error", "undefined return code from storage"))
		http.Error(res, "request of withdraw process error", http.StatusInternalServerError)
		return
	}
}

func Withdrawals(res http.ResponseWriter, req *http.Request, wthdrwls repositories.WithdrawalsInterface) {
	// Получаю id пользователя из контекста
	id, ok := req.Context().Value(auth.UserIDKey).(string)
	if !ok {
		logger.ServerLog.Error("user ID not found", zap.String("address", req.URL.String()))
		http.Error(res, "user ID not found", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	defer req.Body.Close()

	withdrawals, code, err := wthdrwls.GetWithdrawals(req.Context(), id)
	if err != nil {
		logger.ServerLog.Error("get withdrawals error", zap.String("address", req.URL.String()), zap.String("error", err.Error()))
		http.Error(res, "get withdrawals error", http.StatusInternalServerError)
		return
	}

	switch code {
	case repositories.WITHDRAWALS200:
		// устанавливаю заголовок таким образом вместо WriteHeader(http.StatusOK), потому что
		// далее в методе Write в middleware необходимо установить заголовок Hash со значением хэша,
		// а после WriteHeader заголовки уже не устанавливаются
		res.Header().Set("Status-Code", "200")
		enc := json.NewEncoder(res)
		if err := enc.Encode(withdrawals); err != nil {
			logger.ServerLog.Error("error encoding response", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	case repositories.WITHDRAWALS204:
		res.WriteHeader(http.StatusNoContent)
		return
	}
}

// -----------------------------------------------------------------------------------------------------------------------------------------------

func NotFoundHandler() http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		NotFound(res, req)
	}
	return fn
}

func RegisterHandler(regist repositories.AuthInterface) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		Register(res, req, regist)
	}
	return fn
}

func AuthenticationHandler(regist repositories.AuthInterface) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		Authentication(res, req, regist)
	}
	return fn
}

func LoadOrdersHandler(order repositories.OrdersInterface) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		LoadOrders(res, req, order)
	}
	return fn
}

func GetOrdersHandler(order repositories.OrdersInterface) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		GetOrders(res, req, order)
	}
	return fn
}

func GetBalanceHandler(balance repositories.BalanceInterface) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		GetBalance(res, req, balance)
	}
	return fn
}

func WithdrawHandler(withdraw repositories.WithdrawInterface) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		Withdraw(res, req, withdraw)
	}
	return fn
}

func WithdrawalsHandler(wthdrwls repositories.WithdrawalsInterface) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		Withdrawals(res, req, wthdrwls)
	}
	return fn
}
