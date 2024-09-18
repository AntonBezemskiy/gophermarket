package pg

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrdersForAccrual(t *testing.T) {
	// тесты с базой данных
	// предварительно необходимо создать тестовую БД и определить параметры сединения host=host user=user password=password dbname=dbname  sslmode=disable
	{
		// инициализация базы данных-------------------------------------------------------------------
		databaseDsn := "host=localhost user=testgophermartpg password=newpassword dbname=testgophermartpg sslmode=disable"

		// создаём соединение с СУБД PostgreSQL
		conn, err := sql.Open("pgx", databaseDsn)
		require.NoError(t, err)
		defer conn.Close()

		// Проверка соединения с БД
		ctx := context.Background()
		err = conn.PingContext(ctx)
		require.NoError(t, err)

		// создаем экземпляр хранилища pg
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		//-------------------------------------------------------------------------------------------------------------

		// проверка успешной выгрузки
		{
			//предварительно загружаю в базу заказы-------------------------
			status, err := stor.Load(ctx, "GetOrdersForAccrual one", "731447180373")
			require.NoError(t, err)
			assert.Equal(t, 202, status)
			//устанавливаю паузу, чтобы заказы имели разное время загрузки
			time.Sleep(100 * time.Millisecond)

			status, err = stor.Load(ctx, "user id one pg", "250788087147")
			require.NoError(t, err)
			assert.Equal(t, 202, status)
			time.Sleep(100 * time.Millisecond)

			status, err = stor.Load(ctx, "user id two pg", "442338022134")
			require.NoError(t, err)
			assert.Equal(t, 202, status)
			time.Sleep(100 * time.Millisecond)

			numbers, err := stor.GetOrdersForAccrual(ctx)
			require.NoError(t, err)
			assert.Equal(t, 3, len(numbers))

			for i, number := range numbers {
				if i == 0 {
					numWant := int64(731447180373)
					assert.Equal(t, numWant, number)
				}
				if i == 1 {
					numWant := int64(250788087147)
					assert.Equal(t, numWant, number)
				}
				if i == 2 {
					numWant := int64(442338022134)
					assert.Equal(t, numWant, number)
				}
			}
		}

		// Удаление данных из тестовых таблиц для выполнения следующих тестов------------------------------------------
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
}

func TestGetRetryPeriod(t *testing.T) {
	// тесты с базой данных
	// предварительно необходимо создать тестовую БД и определить параметры сединения host=host user=user password=password dbname=dbname  sslmode=disable
	{
		// инициализация базы данных-------------------------------------------------------------------
		databaseDsn := "host=localhost user=testgophermartpg password=newpassword dbname=testgophermartpg sslmode=disable"

		// создаём соединение с СУБД PostgreSQL
		conn, err := sql.Open("pgx", databaseDsn)
		require.NoError(t, err)
		defer conn.Close()

		// Проверка соединения с БД
		ctx := context.Background()
		err = conn.PingContext(ctx)
		require.NoError(t, err)

		// создаем экземпляр хранилища pg
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		//-------------------------------------------------------------------------------------------------------------

		// тестовый случай 1
		tOne := time.Now().UTC().Truncate(time.Second) // Обрезаем наносекунды
		err = stor.AddRetryPeriod(ctx, "accrual", tOne)
		require.NoError(t, err)
		getOne, err := stor.GetRetryPeriod(ctx, "accrual")
		require.NoError(t, err)
		assert.Equal(t, tOne, getOne)

		// тестовый случай 2
		tTwo := time.Now().Add(60 * time.Second).UTC().Truncate(time.Second) // Обрезаем наносекунды
		err = stor.AddRetryPeriod(ctx, "accrual", tTwo)
		require.NoError(t, err)
		getTwo, err := stor.GetRetryPeriod(ctx, "accrual")
		require.NoError(t, err)
		assert.Equal(t, tTwo, getTwo)

		// тестовый случай 3, случай с ошибкой
		_, err = stor.GetRetryPeriod(ctx, "wrong service")
		require.Error(t, err)

		// Удаление данных из тестовых таблиц для выполнения следующих тестов------------------------------------------
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
}
