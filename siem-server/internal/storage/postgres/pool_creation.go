package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

//Создаём функцию создания нового пула соединений с БД
//принимает строку в которой содержится ссылка на подключение к БД
//возвращает пул соединений или ошибку, если соединение не удалось

func NewPool(connString string) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), connString)
}
