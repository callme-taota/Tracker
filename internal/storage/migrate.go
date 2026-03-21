package storage

import (
	"database/sql"
	"strings"
)

// migrate applies additive schema changes for existing databases (CREATE TABLE IF NOT EXISTS does not add columns).
func migrate(conn *sql.DB) error {
	stmts := []string{
		`ALTER TABLE jobs ADD COLUMN claimed_by TEXT`,
		`ALTER TABLE jobs ADD COLUMN claimed_at DATETIME`,
	}
	for _, q := range stmts {
		if _, err := conn.Exec(q); err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate column") {
				continue
			}
			return err
		}
	}
	return nil
}
