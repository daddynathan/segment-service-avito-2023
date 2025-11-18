package repository

import (
	"database/sql"
	"fmt"
	"os"
)

func RunMigrations(db *sql.DB) error {
	sqlContent, err := os.ReadFile("./internal/repository/migrations/schema.sql")
	if err != nil {
		return fmt.Errorf("error reading schema.sql: %w", err)
	}
	_, err = db.Exec(string(sqlContent))
	if err != nil {
		return fmt.Errorf("error executing schema.sql: %w", err)
	}
	return nil
}
