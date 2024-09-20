package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/AntonBezemskiy/gophermart/internal/accrual"
	"github.com/AntonBezemskiy/gophermart/internal/auth"
	"github.com/AntonBezemskiy/gophermart/internal/handlers"
	"github.com/AntonBezemskiy/gophermart/internal/logger"
	"github.com/AntonBezemskiy/gophermart/internal/pg"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

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

	// в целях дебага удаляю данные из базы
	_ = stor.Disable(ctx)

	if err := run(ctx, stor); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
}

// функция run будет необходима для инициализации зависимостей сервера перед запуском
func run(ctx context.Context, stor *pg.Store) error {
	if err := logger.Initialize(logLevel); err != nil {
		return err
	}

	logger.ServerLog.Info("Running gophermart", zap.String("address", netAddr))
	// запускаю отправку запросов к системе расчета баллов accrual в отдельной горутине
	go accrual.UpdateAccrualData(ctx, stor, stor)
	// запускаю сам сервис
	return http.ListenAndServe(netAddr, MetricRouter(stor))
}

func MetricRouter(stor *pg.Store) chi.Router {
	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", logger.RequestLogger(handlers.RegisterHandler(stor)))
		r.Post("/login", logger.RequestLogger(handlers.AuthenticationHandler(stor)))
		r.Post("/orders", logger.RequestLogger(auth.Checker(handlers.LoadOrdersHandler(stor))))
		r.Get("/orders", logger.RequestLogger(auth.Checker(handlers.GetOrdersHandler(stor))))

		r.Route("/balance", func(r chi.Router) {
			r.Get("/", logger.RequestLogger(auth.Checker(handlers.GetBalanceHandler(stor))))
			r.Post("/withdraw", logger.RequestLogger(auth.Checker(handlers.WithdrawHandler(stor))))
		})
		r.Get("/withdrawals", logger.RequestLogger(auth.Checker(handlers.WithdrawalsHandler(stor))))
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(logger.RequestLogger(handlers.NotFoundHandler()))

	return r
}
