package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/AntonBezemskiy/gophermart/internal/auth"
	"github.com/AntonBezemskiy/gophermart/internal/logger"
	"github.com/AntonBezemskiy/gophermart/internal/repositories"
	"github.com/AntonBezemskiy/gophermart/internal/tools"
	"go.uber.org/zap"
)

// Store реализует интерфейс store.Store и позволяет взаимодействовать с СУБД PostgreSQL
type Store struct {
	// Поле conn содержит объект соединения с СУБД
	conn *sql.DB
}

// NewStore возвращает новый экземпляр PostgreSQL-хранилища
func NewStore(conn *sql.DB) *Store {
	return &Store{conn: conn}
}

func (s Store) Bootstrap(ctx context.Context) (err error) {
	// запускаем транзакцию
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// откат транзакции в случае ошибки
	defer tx.Rollback()

	// создаю таблицу для хранения данных пользователя. ----------------------------------------------
	_, err = tx.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS auth (
            login varchar(128) PRIMARY KEY,
            password varchar(128),
            token varchar(256)
        )
    `)
	if err != nil {
		return err
	}
	// создаю уникальный индекс для логина
	_, err = tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS login ON auth (login)`)
	if err != nil {
		return err
	}

	// создаю таблицу для хранения заказов-------------------------------------------------------------
	_, err = tx.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS orders (
			number bigint PRIMARY KEY,
			status varchar(128),
			accrual double precision,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			id_user varchar(128)
        )
    `)
	// создаю индекс для статуса
	_, err = tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS status ON orders (status)`)
	if err != nil {
		return err
	}
	// создаю индекс для id пользователя
	_, err = tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS id_user ON orders (id_user)`)
	if err != nil {
		return err
	}

	// создаю таблицу для хранения баланса пользователя-------------------------------------------------------------
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS balance (
			id_user varchar(128) PRIMARY KEY,
			current double precision,
			withdrawn double precision
		)
	`)
	// создаю уникальный индекс для статуса
	_, err = tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS id_user ON balance (id_user)`)
	if err != nil {
		return err
	}

	// создаю таблицу для хранения информации о выводе средств
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS withdrawals (
			number bigint PRIMARY KEY,
			withdrawn double precision,
			processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			id_user varchar(128)
		)
	`)
	// создаю уникальный индекс номера заказа
	_, err = tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS number ON withdrawals (number)`)
	if err != nil {
		return err
	}

	// создаю таблицу для хранения информации периоде времени в секундах, в течении которого сервис не должен отправлять запросы к accrual
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS wait (
			service varchar(128) PRIMARY KEY,
			period TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)

	// коммитим транзакцию
	err = tx.Commit()
	return err
}

// Disable очищает БД, удаляя записи из таблиц
// необходима для тестирования, чтобы в процессе удалять тестовые записи
func (s Store) Disable(ctx context.Context) (err error) {
	logger.ServerLog.Debug("truncate all data in all tables")

	// запускаем транзакцию
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// в случае неуспешного коммита все изменения транзакции будут отменены
	defer tx.Rollback()

	// удаляю все записи в таблице auth
	_, err = tx.ExecContext(ctx, `
			TRUNCATE TABLE auth 
	`)
	if err != nil {
		return err
	}

	// удаляю все записи в таблице orders
	_, err = tx.ExecContext(ctx, `
			TRUNCATE TABLE orders 
	`)
	if err != nil {
		return err
	}

	// удаляю все записи в таблице balance
	_, err = tx.ExecContext(ctx, `
			TRUNCATE TABLE balance 
	`)
	if err != nil {
		return err
	}

	// удаляю все записи в таблице withdrawals
	_, err = tx.ExecContext(ctx, `
			TRUNCATE TABLE withdrawals 
	`)
	if err != nil {
		return err
	}

	// удаляю все записи в таблице wait
	_, err = tx.ExecContext(ctx, `
			TRUNCATE TABLE wait
	`)
	if err != nil {
		return err
	}

	// коммитим транзакцию
	return tx.Commit()
}

// Регистрация пользователя в системе. Создаю таблицу с учетными данными пользователя, а так-же таблицу с балансом пользователя
func (s Store) Register(ctx context.Context, login string, password string) (ok bool, token string, err error) {
	// проверка валидности логина и пароля
	// наверное лучше вместо ошибки возвращать статус вроде http.StatusBadRequest
	if login == "" || password == "" {
		err = fmt.Errorf("login or password is invalid")
		return
	}

	query := `
		SELECT login
		FROM auth
		WHERE login = $1
	`
	// Проверяю уникальность логина
	// корректность пароля не проверяю
	row := s.conn.QueryRowContext(ctx, query, login)
	var checkUniqueOfLoggin string
	err = row.Scan(&checkUniqueOfLoggin)
	if err != nil {
		if err == sql.ErrNoRows {
			// логин уникален, продолжаем регистрацию
			logger.ServerLog.Debug("user is unique, continue registration process", zap.String("login", login), zap.String("password", password))
			err = nil
		} else {
			// Ошибка метода Scan
			return
		}
	} else {
		// пользователь не уникален
		ok = false
		return
	}

	// генерирую id нового пользователя, которое будет упаковано в веб токен
	token, err = auth.BuildJWTString(24)
	if err != nil {
		return
	}

	registerUser := `
				INSERT INTO auth (login, password, token)
				VALUES ($1, $2, $3);
				`
	_, err = s.conn.ExecContext(ctx, registerUser, login, password, token)
	if err != nil {
		return
	}

	// создаю в таблице balance запись о балансе нового пользователя
	// для этого получаю id пользователя из сгенерированного токена
	id, errID := auth.GetUserID(token)
	if errID != nil {
		err = errID
		return
	}
	createBalanceData := `
				INSERT INTO balance (id_user, current, withdrawn)
				VALUES ($1, $2, $3);
				`
	_, err = s.conn.ExecContext(ctx, createBalanceData, id, 0, 0)
	if err != nil {
		return
	}

	ok = true
	return
}

func (s Store) Authenticate(ctx context.Context, login string, password string) (ok bool, token string, err error) {
	// проверка валидности логина и пароля
	// наверное лучше вместо ошибки возвращать статус вроде http.StatusBadRequest
	if login == "" || password == "" {
		err = fmt.Errorf("login or password is invalid")
		return
	}

	query := `
		SELECT
			password,
			token
		FROM auth
		WHERE login = $1
	`
	// Проверяю, что пользователь зарегистрирован в системе и получаю токен пользователя
	row := s.conn.QueryRowContext(ctx, query, login)
	var passwordFromDB string
	err = row.Scan(&passwordFromDB, &token)
	if err != nil {
		if err == sql.ErrNoRows {
			// логин уникален, пользователь не зарегистрирован. Вместо ошибки возвращаю соотвествующий статус
			err = nil
			ok = false
			return
		} else {
			// Ошибка метода Scan
			return
		}
	}

	// проверяю указанный пароль на соотвествие с тем, что хранится в базе
	if password != passwordFromDB {
		ok = false
		return
	}
	ok = true
	return
}

func (s Store) Load(ctx context.Context, idUser string, orderNumber string) (status int, err error) {
	check := tools.LuhnCheck(orderNumber)
	if !check {
		status = repositories.ORDERSCODE422
		return
	}

	// Преобразую номер заказа из строки в int64
	orderNumberInt, err := strconv.ParseInt(orderNumber, 10, 64)
	if err != nil {
		// переданный параметр не является числом, возвращаю соответствующий статус
		err = nil
		status = repositories.WITHDRAWCODE422
		return
	}

	query := `
		SELECT
			id_user
		FROM orders
		WHERE number = $1
	`
	// проверяю, что заказ с данным номером ещё не был добавлен в систему
	row := s.conn.QueryRowContext(ctx, query, orderNumberInt)
	var idFromDB string
	err = row.Scan(&idFromDB)
	if err != nil {
		if err == sql.ErrNoRows {
			// заказ ещё не был загружен в систему, можно продолжать процесс загрузки
			err = nil
		} else {
			// Ошибка метода Scan
			return
		}
	} else {
		if idFromDB == idUser {
			// заказ уже загружен этим пользователем
			status = repositories.ORDERSCODE200
			return
		} else {
			// заказ был загружен другим пользователем
			status = repositories.ORDERSCODE409
			return
		}
	}
	registerUser := `
				INSERT INTO orders (number, status, accrual, id_user)
				VALUES ($1, $2, $3, $4);
				`
	_, err = s.conn.ExecContext(ctx, registerUser, orderNumberInt, repositories.NEW, 0, idUser)
	if err != nil {
		return
	}
	status = repositories.ORDERSCODE202
	return
}

func (s Store) GetOrders(ctx context.Context, idUser string) (orders []repositories.Order, status int, err error) {
	orders = make([]repositories.Order, 0)

	// выгружаю все заказы, которые соответствуют данному польззователя. Сортировка по времени от самых новых к самым старым заказам
	query := `
		SELECT
			number,
			status,
			accrual,
			uploaded_at
		FROM orders
		WHERE id_user = $1
		ORDER BY uploaded_at DESC
	`
	// получаю заказы, которые данный пользователь загрузил в систему
	rows, errQuery := s.conn.QueryContext(ctx, query, idUser)
	if errQuery != nil {
		err = errQuery
		return
	}
	// обязательно закрываем перед возвратом функции
	defer rows.Close()

	ordersExist := false
	// собираю заказы для ответа сервера
	for rows.Next() {
		// флаг, что пользователь загрузил как минимум один заказ
		ordersExist = true

		var order repositories.Order
		err = rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return
		}
		orders = append(orders, order)
	}
	// проверяю, есть ли у данного пользователя заказы
	if !ordersExist {
		//пользователя не загрузил ни одного заказа, возвращаю статус 204
		status = repositories.GETORDERSCODE204
		return
	}
	// проверяю на ошибки
	err = rows.Err()
	if err != nil {
		return
	}
	status = repositories.GETORDERSCODE200
	return
}

// Метод для получения id пользователя, который загрузил данный заказ
func (s Store) GetIDByOrderNumber(ctx context.Context, orderNumber string) (string, error) {
	// Преобразую номер заказа из строки в int64
	orderNumberInt, err := strconv.ParseInt(orderNumber, 10, 64)
	if err != nil {
		return "", fmt.Errorf("error of converting string to int64: %w", err)
	}

	query := `
		SELECT
			id_user
		FROM orders
		WHERE number = $1
	`
	// получаю id пользователя по номеру заказа, чтобы обновить баланс именно того пользователя, который загрузил заказа
	row := s.conn.QueryRowContext(ctx, query, orderNumberInt)
	var idFromDB string
	err = row.Scan(&idFromDB)
	if err != nil {
		if err == sql.ErrNoRows {
			// заказ ещё не был загружен в систему, такая ситуация является внутренней ошибкой сервиса
			return "", fmt.Errorf("internal error: order is not loaded in gophermart %w", err)
		} else {
			// Ошибка метода Scan
			return "", err
		}
	}
	return idFromDB, nil
}

// Обновляет баланс пользователя по номеру заказа и сумме начисленных баллов
func (s Store) UpdateBalance(ctx context.Context, orderNumber string, accrual float64) error {
	// Преобразую номер заказа из строки в int64
	orderNumberInt, err := strconv.ParseInt(orderNumber, 10, 64)
	if err != nil {
		return fmt.Errorf("error of converting string to int64: %w", err)
	}

	query := `
		SELECT
			id_user
		FROM orders
		WHERE number = $1
	`
	// получаю id пользователя по номеру заказа, чтобы обновить баланс именно того пользователя, который загрузил заказа
	row := s.conn.QueryRowContext(ctx, query, orderNumberInt)
	var idFromDB string
	err = row.Scan(&idFromDB)
	if err != nil {
		if err == sql.ErrNoRows {
			// заказ ещё не был загружен в систему, такая ситуация является внутренней ошибкой сервиса
			return fmt.Errorf("internal error: order is not loaded in gophermart %w", err)
		} else {
			// Ошибка метода Scan
			return err
		}
	}

	updateBalance := `
				UPDATE balance
				SET current = current + $1
				WHERE id_user = $2;
		`
	_, err = s.conn.ExecContext(ctx, updateBalance, accrual, idFromDB)

	return err
}

// обновляю инофрмацию в заказах. Так же в случае начисления баллов обновляю баланс пользователя
func (s Store) UpdateOrder(ctx context.Context, orderNumber string, status string, accrual float64) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// откат транзакции в случае ошибки
	defer tx.Rollback()

	updateOrder := `
				UPDATE orders
				SET status = $1,
    				accrual = accrual + $2
				WHERE number = $3;
	`
	_, err = tx.ExecContext(ctx, updateOrder, status, accrual, orderNumber)
	if err != nil {
		return err
	}
	// обновляю баланс пользователя если accrula успешно обработал заказ и были начислены баллы
	if status == repositories.PROCESSED {
		// получаю id пользователя по номеру заказа
		id, err := s.GetIDByOrderNumber(ctx, orderNumber)
		if err != nil {
			return err
		}
		// обновляю баланс пользователя в рамках одной транзакции
		updateBalance := `
				UPDATE balance
				SET current = current + $1
				WHERE id_user = $2;
		`
		_, err = tx.ExecContext(ctx, updateBalance, accrual, id)
		if err != nil {
			return err
		}
	}

	// коммитим транзакцию
	return tx.Commit()
}

// обновляю инофрмацию в заказах транзакцией
func (s Store) UpdateOrderTX(ctx context.Context, dataSlice []repositories.AccrualData) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// откат транзакции в случае ошибки
	defer tx.Rollback()

	updateOrder := `
				UPDATE orders
				SET status = $1,
    				accrual = accrual + $2
				WHERE number = $3;
	`

	for _, data := range dataSlice {
		_, err := tx.ExecContext(ctx, updateOrder, data.Status, data.Accrual, data.Order)
		if err != nil {
			return err
		}
		// обновляю баланс пользователя если accrula успешно обработал заказ и были начислены баллы
		if data.Status == repositories.PROCESSED {
			// получаю id пользователя по номеру заказа
			id, err := s.GetIDByOrderNumber(ctx, data.Order)
			if err != nil {
				return err
			}
			// обновляю баланс пользователя в рамках одной транзакции
			updateBalance := `
				UPDATE balance
				SET current = current + $1
				WHERE id_user = $2;
			`
			_, err = tx.ExecContext(ctx, updateBalance, data.Accrual, id)
			if err != nil {
				return err
			}
		}
	}

	// коммитим транзакцию
	return tx.Commit()
}

// выгружаю номера заказов для которых необходимо получить баллы лояльности
func (s Store) GetOrdersForAccrual(ctx context.Context) (numbers []int64, err error) {
	numbers = make([]int64, 0)

	// выгружаю все заказы со статусами NEW и PROCESSING. Сортировка по времени от самых старых к самым новым заказам
	query := `
		SELECT
			number
		FROM orders
		WHERE status = $1 OR status = $2
		ORDER BY uploaded_at
	`
	rows, errQuery := s.conn.QueryContext(ctx, query, repositories.NEW, repositories.PROCESSING)
	if errQuery != nil {
		err = errQuery
		return
	}
	// обязательно закрываем перед возвратом функции
	defer rows.Close()

	// собираю номера заказов
	for rows.Next() {
		var number int64
		err = rows.Scan(&number)
		if err != nil {
			return
		}
		numbers = append(numbers, number)
	}
	// проверяю на ошибки
	err = rows.Err()
	if err != nil {
		return
	}
	return
}

func (s Store) GetBalance(ctx context.Context, idUser string) (balance repositories.Balance, err error) {
	query := `
		SELECT
			current,
			withdrawn
		FROM balance
		WHERE id_user = $1
	`
	// получаю баланс пользователя
	row := s.conn.QueryRowContext(ctx, query, idUser)
	err = row.Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		// запись о балансе пользователя создается сразу при регистрации, поэтому отсутствие записи это внутренняя ошибка
		return
	}
	return
}

// Запрос на списание средств. Так же в методе обновляется информация о выводе средств
func (s Store) Withdraw(ctx context.Context, idUser string, orderNumber string, sum float64) (status int, err error) {
	// Проверяю кооректность номера заказа. Номер заказа некорректный в случае если он не проходит проверку по алгоритму Луна
	// или если такой заказ уже существует в сервисе
	check := tools.LuhnCheck(orderNumber)
	if !check {
		status = repositories.WITHDRAWCODE422
		return
	}

	// Преобразую номер заказа из строки в int64
	orderNumberInt, err := strconv.ParseInt(orderNumber, 10, 64)
	if err != nil {
		// переданный параметр не является числом, возвращаю соответствующий статус
		err = nil
		status = repositories.WITHDRAWCODE422
		return
	}

	query := `
		SELECT
			id_user
		FROM orders
		WHERE number = $1
	`
	// проверяю, что заказ с данным номером ещё не был добавлен в систему
	row := s.conn.QueryRowContext(ctx, query, orderNumberInt)
	var idFromDB string
	err = row.Scan(&idFromDB)
	if err != nil {
		if err == sql.ErrNoRows {
			// заказ ещё не был загружен в систему, можно продолжать процесс загрузки
			err = nil
		} else {
			// Ошибка метода Scan
			return
		}
	} else {
		// заказ уже добавлен в систему
		status = repositories.WITHDRAWCODE422
		return
	}
	//------------------------------------------------------------------------------------

	// проверяю баланс пользователя--------------------------------------------------------

	balance, err := s.GetBalance(ctx, idUser)
	if err != nil {
		return
	}
	if balance.Current < sum {
		// на счету недостаточно средств
		status = repositories.WITHDRAWCODE402
		return
	}

	// вношу изменения в базу данных. Обновляю баланс пользователя и таблицу с информацией о выводе средств
	// выполняю эти операции внутри одной транзакции

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	// откат транзакции в случае ошибки
	defer tx.Rollback()

	updateBalance := `
		UPDATE balance
		SET current = current - $1,
    		withdrawn = withdrawn + $2
		WHERE id_user = $3;
	`

	// обновляю баланс пользователя
	_, err = tx.ExecContext(ctx, updateBalance, sum, sum, idUser)

	// добавляю запись в таблицу с информацией о выводе средств
	insertWithdrawn := `
				INSERT INTO withdrawals (number, withdrawn, id_user)
				VALUES ($1, $2, $3);
				`
	_, err = tx.ExecContext(ctx, insertWithdrawn, orderNumberInt, sum, idUser)
	if err != nil {
		return
	}

	// Устанавливаю статус успешной обработки запроса
	status = repositories.WITHDRAWCODE200

	// коммитим транзакцию
	err = tx.Commit()
	return
}

func (s Store) GetWithdrawals(ctx context.Context, idUser string) (withdrawals []repositories.Withdrawals, status int, err error) {
	withdrawals = make([]repositories.Withdrawals, 0)

	// получаю записи о выводе средств для данного пользователя
	query := `
		SELECT
			number,
			withdrawn,
			processed_at
		FROM withdrawals
		WHERE id_user = $1
		ORDER BY processed_at DESC
	`
	rows, errQuery := s.conn.QueryContext(ctx, query, idUser)
	if errQuery != nil {
		err = errQuery
		return
	}
	// обязательно закрываем перед возвратом функции
	defer rows.Close()

	withdrawnExist := false
	// собираю заказы для ответа сервера
	for rows.Next() {
		// флаг, что есть как минимум одно списание средств
		withdrawnExist = true

		var w repositories.Withdrawals
		err = rows.Scan(&w.Order, &w.Sum, &w.ProcessAt)
		if err != nil {
			return
		}
		withdrawals = append(withdrawals, w)
	}
	// проверяю, есть ли у данного пользователя списания
	if !withdrawnExist {
		// нет ни одного списания
		status = repositories.WITHDRAWALS204
		return
	}
	// проверяю на ошибки
	err = rows.Err()
	if err != nil {
		return
	}
	status = repositories.WITHDRAWALS200
	return
}

// устанавливаю период ожидания для определенного сервиса
func (s Store) AddRetryPeriod(ctx context.Context, service string, period time.Time) error {
	// добавляю запись в таблицу с информацией о выводе средств
	insertPeriod := `
				INSERT INTO wait (service, period)
				VALUES ($1, $2)
				ON CONFLICT (service)
				DO UPDATE SET period = EXCLUDED.period;
				`
	_, err := s.conn.ExecContext(ctx, insertPeriod, service, period)
	if err != nil {
		return err
	}
	return nil
}

// получаю период ожидания для определенного сервиса
func (s Store) GetRetryPeriod(ctx context.Context, service string) (time.Time, error) {
	query := `
		SELECT
			period
		FROM wait
		WHERE service = $1
	`

	row := s.conn.QueryRowContext(ctx, query, service)
	var period time.Time
	err := row.Scan(&period)
	if err != nil {
		return time.Now(), err
	}
	return period, nil
}
