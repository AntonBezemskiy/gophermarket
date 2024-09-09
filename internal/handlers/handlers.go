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
	if !ok {
		logger.ServerLog.Error("register of new user is failed", zap.String("address", req.URL.String()), zap.String("error", "login is not unique"))
		http.Error(res, "login is not unique", http.StatusConflict)
		return
	}
	if err != nil {
		logger.ServerLog.Error("register new user error", zap.String("address", req.URL.String()), zap.String("error: ", error.Error(err)))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Для передачи аутентификационных данных использую механизм cookie
	auth.SetTokenCookie(res, token)
	res.WriteHeader(200)
}

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
	if !ok {
		logger.ServerLog.Error("authentication of user is failed", zap.String("address", req.URL.String()), zap.String("error", "login or password is wrong"))
		http.Error(res, "login or password is wrong", http.StatusUnauthorized)
		return
	}
	if err != nil {
		logger.ServerLog.Error("authentication of user error", zap.String("address", req.URL.String()), zap.String("error: ", error.Error(err)))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Для передачи аутентификационных данных использую механизм cookie
	auth.SetTokenCookie(res, token)
	res.WriteHeader(200)
}

func LoadOrders(res http.ResponseWriter, req *http.Request, order repositories.OredersInterface) {
	res.Header().Set("Content-Type", "text/plain")
	defer req.Body.Close()

	strOrderNumd, err := io.ReadAll(req.Body)
	if err != nil {
		logger.ServerLog.Error("can't get order number in LoadOrders", zap.String("address", req.URL.String()), zap.String("error", err.Error()))
		http.Error(res, "can't get order number in LoadOrders", http.StatusBadRequest)
		return
	}

	//fmt.Printf("\ntest print: %s\n", string(strOrderNumd))

	code, err := order.Load(req.Context(), string(strOrderNumd))
	if err != nil {
		//fmt.Printf("\nerror in Load calling, err: %s\n", err.Error())

		logger.ServerLog.Error("load order error", zap.String("address", req.URL.String()), zap.String("error", err.Error()))
		http.Error(res, "load order error", http.StatusInternalServerError)
		return
	}

	switch code {
	case repositories.ORDERSCODE200:
		res.WriteHeader(200)
		return
	case repositories.ORDERSCODE202:
		res.WriteHeader(202)
		return
	case repositories.ORDERSCODE409:
		res.WriteHeader(409)
		return
	case repositories.ORDERSCODE422:
		res.WriteHeader(422)
		return
	default:
		//fmt.Printf("\nundefined return code from storage, code: %d\n", code)

		logger.ServerLog.Error("load order error", zap.String("address", req.URL.String()), zap.String("error", "undefined return code from storage"))
		http.Error(res, "load order error", http.StatusInternalServerError)
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

func LoadOrdersHandler(order repositories.OredersInterface) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		LoadOrders(res, req, order)
	}
	return fn
}
