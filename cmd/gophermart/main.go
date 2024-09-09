package main

import (
	"log"
	"net/http"

	"github.com/AntonBezemskiy/gophermart/internal/auth"
	"github.com/AntonBezemskiy/gophermart/internal/handlers"
	"github.com/AntonBezemskiy/gophermart/internal/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	parseFlags()

	if err := run(); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
}

// функция run будет необходима для инициализации зависимостей сервера перед запуском
func run() error {
	if err := logger.Initialize(logLevel); err != nil {
		return err
	}

	logger.ServerLog.Info("Running gophermart", zap.String("address", netAddr))
	return http.ListenAndServe(netAddr, MetricRouter())
}

func MetricRouter() chi.Router {
	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", logger.RequestLogger(handlers.RegisterHandler(nil)))
		r.Post("/login", logger.RequestLogger(handlers.AuthenticationHandler(nil)))
		r.Post("/orders", logger.RequestLogger(auth.Checker(handlers.LoadOrdersHandler(nil))))
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(logger.RequestLogger(handlers.NotFoundHandler()))

	return r
}
