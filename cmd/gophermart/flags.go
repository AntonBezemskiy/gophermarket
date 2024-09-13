package main

import (
	"flag"
	"os"
)

var (
	netAddr      string
	logLevel string
	databaseDsn     string
	accrualSystemAddress  string
)

// конфигурирование приложения с приоритетом у значений флагов
func parseFlags() {
	var flagNetAddr string
	flag.StringVar(&flagNetAddr, "a", "", "address and port to run server")

	// настройка флага для хранения метрик в базе данных
	var flagDatabaseDsn     string
	flag.StringVar(&flagDatabaseDsn, "d", "", "database connection address") // host=localhost user=metrics password=metrics dbname=metricsdb  sslmode=disable

	// настройка адреса системы расчёта начислений через флаг
	var flagAccrualSystemAddress string
	flag.StringVar(&flagAccrualSystemAddress, "r", "", "address of connection to accrual system")

	var flagLogLevel string
	flag.StringVar(&flagLogLevel, "l", "info", "log level")

	flag.Parse()

	// устанавливаю значения равными соответствующим переменным окружения, в случае если не задан флаг и если существует 
	// переменная окружения с непустым значением
	if flagNetAddr == "" {
		if envRunAddr := os.Getenv("RUN_ADDRESS"); envRunAddr != "" {
			flagNetAddr = envRunAddr
		}
	}
	if flagDatabaseDsn == ""{
		if envDatabaseDsn := os.Getenv("DATABASE_URI"); envDatabaseDsn != "" {
			flagDatabaseDsn = envDatabaseDsn
		}
	}
	if flagAccrualSystemAddress == ""{
		if envAccrualSystemAddress := os.Getenv("DATABASE_URI"); envAccrualSystemAddress != "" {
			flagDatabaseDsn = envAccrualSystemAddress
		}
	}
	if flagLogLevel == ""{
		if envLogLevel := os.Getenv("DATABASE_URI"); envLogLevel != "" {
			flagLogLevel = envLogLevel
		}
	}

	// устанавливаю значения по умолчанию, если значения не были заданы ни флагом, ни переменной окружения
	if flagNetAddr == ""{
		netAddr = ":8080"
	}
	if flagDatabaseDsn == ""{
		databaseDsn = ""
	}
	if flagAccrualSystemAddress == ""{
		accrualSystemAddress = ""
	}
	if flagLogLevel == ""{
		logLevel = ""
	}
}
