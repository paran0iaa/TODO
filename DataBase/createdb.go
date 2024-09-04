package database

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/paran0iaa/TODO/internal/services"

	_ "github.com/mattn/go-sqlite3"
)

func CreateDb(dbName string) {
	db, err := sql.Open("sqlite3", services.GetEnv("TODO_DBFILE"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(services.GetEnv("TODO_DBFILE")); os.IsNotExist(err) {
		_, err = db.ExecContext(context.Background(),
			`CREATE TABLE IF NOT EXISTS scheduler (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                date TEXT NOT NULL,
                title TEXT NOT NULL,
                comment TEXT,
                repeat TEXT CHECK(length(repeat) <= 128)
            );`)

		if err != nil {
			log.Fatalf("failed to create table: %v", err)
		}

		_, err = db.ExecContext(context.Background(),
			`CREATE INDEX IF NOT EXISTS scheduler_date ON scheduler (date);`)
		if err != nil {
			log.Fatalf("failed to create index: %v", err)
		}
	}
}
