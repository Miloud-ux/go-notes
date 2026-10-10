package main

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(path string) (*SQLStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		// error opening the db
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS notes (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				text TEXT NOT NULL
			)
		`)

	if err != nil {
		return nil, err
	}

	return &SQLStore{db: db}, nil
}
