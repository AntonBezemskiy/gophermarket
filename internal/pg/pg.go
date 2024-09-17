package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/AntonBezemskiy/gophermart/internal/auth"
	"github.com/AntonBezemskiy/gophermart/internal/repositories"
	"github.com/AntonBezemskiy/gophermart/internal/tools"
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
			accrual INTEGER,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			id_user varchar(128)
        )
    `)
	// создаю уникальный индекс для статуса
	_, err = tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS status ON orders (status)`)
	if err != nil {
		return err
	}
	// создаю уникальный индекс для id пользователя
	_, err = tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS id_user ON orders (id_user)`)
	if err != nil {
		return err
	}

	// коммитим транзакцию
	err = tx.Commit()
	return err
}

// Bootstrap останавливает работу БД, удаляя таблицы и индексы
func (s Store) Disable(ctx context.Context) (err error) {
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

	// коммитим транзакцию
	return tx.Commit()
}

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
		err = fmt.Errorf("error of converting string to int64: %w", err)
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

func (s Store) Get(ctx context.Context, idUser string) (orders []repositories.Order, status int, err error) {
	return
}
