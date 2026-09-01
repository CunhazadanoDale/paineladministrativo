package postgres

import (
	"time"

	"github.com/jmoiron/sqlx"
)

func ConnectionDB(connection string) (*sqlx.DB, error) {

	db, err := sqlx.Connect("postgres", connection)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, err
}