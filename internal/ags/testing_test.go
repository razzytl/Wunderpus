package ags

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func tempGoalDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
