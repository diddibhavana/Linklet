package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"linklet/internal/postgres"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" { log.Fatal("DATABASE_URL is required") }
	db, err := sql.Open("pgx", url)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if len(os.Args) != 2 { log.Fatal("usage: migrate up|down") }
	switch os.Args[1] {
	case "up": err = postgres.Migrate(ctx, db)
	case "down": err = postgres.MigrateDown(ctx, db)
	default: log.Fatal("usage: migrate up|down")
	}
	if err != nil { log.Fatal(err) }
}
