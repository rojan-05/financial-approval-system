
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func connectDB() (*sql.DB, error) {
	password := os.Getenv("DB_PASSWORD")

	if password == "" {
		return nil, fmt.Errorf("DB_PASSWORD environment variable is not set")
	}

	connStr := fmt.Sprintf(
		"host=localhost port=5432 user=postgres password=%s dbname=financial_approval sslmode=disable",
		password,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
