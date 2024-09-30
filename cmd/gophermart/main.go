package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AntonBezemskiy/gophermart/internal/accrual"
	"github.com/AntonBezemskiy/gophermart/internal/auth"
	"github.com/AntonBezemskiy/gophermart/internal/handlers"
	"github.com/AntonBezemskiy/gophermart/internal/logger"
	"github.com/AntonBezemskiy/gophermart/internal/pg"
	"github.com/AntonBezemskiy/gophermart/internal/requesttracker"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

const shutdownWaitPeriod = 20 * time.Second // для установки в контекст для реализаации graceful shutdown

func main() {
	parseFlags()

	// Подключение к базе данных
	db, err := sql.Open("pgx", databaseDsn)
	if err != nil {
		log.Fatalf("Error connection to database: %v by address %s", err, databaseDsn)
	}
	defer db.Close()

	// инициализация базы данных-----------------------------------------------------
	// создаём соединение с СУБД PostgreSQL с помощью аргумента командной строки
	conn, err := sql.Open("pgx", databaseDsn)
	if err != nil {
		log.Fatalf("Error create database connection for saving metrics : %v\n", err)
	}

	// Проверка соединения с БД
	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Error checking connection with database: %v\n", err)
	}
	// создаем экземпляр хранилища pg
	stor := pg.NewStore(conn)
	err = stor.Bootstrap(ctx)
	if err != nil {
		log.Fatalf("Error prepare database to work: %v\n", err)
	}
	// ------------------------------------------------------------------------------

	run(ctx, stor)
}

// функция run будет необходима для инициализации зависимостей сервера перед запуском
func run(ctx context.Context, stor *pg.Store) {
	// Инициализация логера
	if err := logger.Initialize(logLevel); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

	logger.ServerLog.Info("Running gophermart", zap.String("address", netAddr))
	// запускаю отправку запросов к системе расчета баллов accrual в отдельной горутине
	go accrual.UpdateAccrualData(ctx, stor, stor)

	// запускаю сам сервис с проверкой отмены контекста для реализации graceful shutdown--------------
	srv := &http.Server{
		Addr:    ":8080",
		Handler: MetricRouter(stor),
	}
	// Канал для получения сигнала прерывания
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Горутина для запуска сервера
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Блокирование до тех пор, пока не поступит сигнал о прерывании
	<-quit
	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownWaitPeriod)
	defer cancel()

	// останавливаю сервер, чтобы он перестал принимать новые запросы
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Stopping server error: %v", err)
	}
	// ожидаю завершения всех активных запросов
	// дополнительный слой защиты и способ отслеживания количества активных запросов на момент поступления сигнала о завершении работы сервера
	requesttracker.WaitForActiveRequests(ctx)

	log.Println("Shutdown the server gracefully")
}

func MetricRouter(stor *pg.Store) chi.Router {
	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", logger.RequestLogger(requesttracker.WithActiveRequests(handlers.RegisterHandler(stor))))
		r.Post("/login", logger.RequestLogger(requesttracker.WithActiveRequests(handlers.AuthenticationHandler(stor))))
		r.Post("/orders", logger.RequestLogger(requesttracker.WithActiveRequests(auth.Checker(handlers.LoadOrdersHandler(stor)))))
		r.Get("/orders", logger.RequestLogger(requesttracker.WithActiveRequests(auth.Checker(handlers.GetOrdersHandler(stor)))))

		r.Route("/balance", func(r chi.Router) {
			r.Get("/", logger.RequestLogger(requesttracker.WithActiveRequests(auth.Checker(handlers.GetBalanceHandler(stor)))))
			r.Post("/withdraw", logger.RequestLogger(requesttracker.WithActiveRequests(auth.Checker(handlers.WithdrawHandler(stor)))))
		})
		r.Get("/withdrawals", logger.RequestLogger(requesttracker.WithActiveRequests(auth.Checker(handlers.WithdrawalsHandler(stor)))))
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(logger.RequestLogger(handlers.NotFoundHandler()))

	return r
}
