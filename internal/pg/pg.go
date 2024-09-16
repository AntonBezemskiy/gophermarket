package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/AntonBezemskiy/gophermart/internal/auth"
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

	// создаю таблицу для хранения данных пользователя.
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
