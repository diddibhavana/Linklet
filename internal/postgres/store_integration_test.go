//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"linklet/internal/domain"
)

func TestCreateAndIdempotentReplay(t *testing.T) {
	dsn:=os.Getenv("DATABASE_URL");if dsn==""{t.Skip("DATABASE_URL is not set")}
	db,err:=sql.Open("pgx",dsn);if err!=nil{t.Fatal(err)};defer db.Close()
	ctx,cancel:=context.WithTimeout(context.Background(),15*time.Second);defer cancel()
	if err=Migrate(ctx,db);err!=nil{t.Fatal(err)}
	store:=&Store{DB:db};now:=time.Now().UTC();key:="integration-"+now.Format("150405.000000000");eventID:="event-"+now.Format("150405000000000000");link:=domain.Link{Code:"it"+now.Format("150405000000"),URL:"https://example.com",CreatedAt:now}
	t.Cleanup(func(){_,_ = db.ExecContext(context.Background(),`DELETE FROM links WHERE code=$1`,link.Code);_,_ = db.ExecContext(context.Background(),`DELETE FROM idempotency_keys WHERE key=$1`,key);_,_ = db.ExecContext(context.Background(),`DELETE FROM outbox WHERE event_id=$1`,eventID)})
	created,err:=store.Create(ctx,link,key,`{"event_id":"`+eventID+`","version":1}`,now);if err!=nil{t.Fatal(err)}
	replay,err:=store.Create(ctx,domain.Link{Code:"other",URL:"https://different.example",CreatedAt:now},key,`{"event_id":"other","version":1}`,now);if err!=nil{t.Fatal(err)}
	if created.ID!=replay.ID||created.Code!=replay.Code{t.Fatalf("idempotency replay mismatch: created=%+v replay=%+v",created,replay)}
}
