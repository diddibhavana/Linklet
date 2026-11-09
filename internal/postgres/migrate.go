package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func Migrate(ctx context.Context, db *sql.DB) error {
	if _,err:=db.ExecContext(ctx,`CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`);err!=nil{return err}
	var applied bool
	if err:=db.QueryRowContext(ctx,`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`,"001_init").Scan(&applied);err!=nil{return err}
	if applied{return nil}
	script,err:=os.ReadFile("migrations/001_init.up.sql")
	if err!=nil{return fmt.Errorf("read migration: %w",err)}
	tx,err:=db.BeginTx(ctx,nil);if err!=nil{return err};defer tx.Rollback()
	for _, statement := range strings.Split(string(script), ";") {
		if strings.TrimSpace(statement) == "" { continue }
		if _, err = tx.ExecContext(ctx, statement); err != nil { return err }
	}
	if _,err=tx.ExecContext(ctx,`INSERT INTO schema_migrations(version) VALUES($1)`,"001_init");err!=nil{return err}
	return tx.Commit()
}

func MigrateDown(ctx context.Context, db *sql.DB) error {
	script, err := os.ReadFile("migrations/001_init.down.sql")
	if err != nil { return fmt.Errorf("read down migration: %w", err) }
	tx, err := db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	for _, statement := range strings.Split(string(script), ";") {
		if strings.TrimSpace(statement) == "" { continue }
		if _, err = tx.ExecContext(ctx, statement); err != nil { return err }
	}
	return tx.Commit()
}
