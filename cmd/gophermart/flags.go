package main

import (
	"flag"
	"log"
	"os"

	"github.com/AntonBezemskiy/gophermart/internal/accrual"
)

var (
	netAddr     string
	logLevel    string
	databaseDsn string
)

// конфигурирование приложения с приоритетом у значений флагов
func parseFlags() {
	var flagNetAddr string
	flag.StringVar(&flagNetAddr, "a", "", "address and port to run server")

	// настройка флага для хранения метрик в базе данных
	var flagDatabaseDsn string
	flag.StringVar(&flagDatabaseDsn, "d", "", "database connection address") // пример: host=localhost user=example password=userpswd dbname=example  sslmode=disable

	// настройка адреса системы расчёта начислений через флаг
	var flagAccrualSystemAddress string
	flag.StringVar(&flagAccrualSystemAddress, "r", "", "address of connection to accrual system")

	var flagLogLevel string
	flag.StringVar(&flagLogLevel, "l", "", "log level")

	flag.Parse()

	// устанавливаю значения равными соответствующим переменным окружения, в случае если не задан флаг и если существует
	// переменная окружения с непустым значением
	if flagNetAddr == "" {
		envRunAddr := os.Getenv("RUN_ADDRESS")
		flagNetAddr = envRunAddr
	}
	if flagDatabaseDsn == "" {
		envDatabaseDsn := os.Getenv("DATABASE_URI")
		flagDatabaseDsn = envDatabaseDsn
	}
	if flagAccrualSystemAddress == "" {
		envAccrualSystemAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS")
		flagAccrualSystemAddress = envAccrualSystemAddress
	}
	if flagLogLevel == "" {
		envLogLevel := os.Getenv("LOG_LEVEL")
		flagLogLevel = envLogLevel
	}

	// устанавливаю значения по умолчанию, если значения не были заданы ни флагом, ни переменной окружения
	if flagNetAddr == "" {
		flagNetAddr = ":8085"
	}
	if flagDatabaseDsn == "" {
		log.Fatalf("DatabaseDsn is not set, set flag -d or system variable DATABASE_URI")
	}
	if flagAccrualSystemAddress == "" {
		flagAccrualSystemAddress = ":8081"
	}
	if flagLogLevel == "" {
		flagLogLevel = "debug"
	}

	// устанавливаю глобальные переменные полученными значениями
	netAddr = flagNetAddr
	databaseDsn = flagDatabaseDsn
	accrual.SetAccrualSystemAddress(flagAccrualSystemAddress)
	logLevel = flagLogLevel
}
