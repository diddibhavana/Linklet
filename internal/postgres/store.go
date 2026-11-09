package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"linklet/internal/domain"
)

type Store struct { DB *sql.DB }

func (s *Store) Create(ctx context.Context, link domain.Link, key, payload string, now time.Time) (domain.Link, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil { return domain.Link{}, err }
	defer tx.Rollback()
	if key != "" {
		if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); err != nil { return domain.Link{}, err }
		var saved []byte
		err = tx.QueryRowContext(ctx, `SELECT response FROM idempotency_keys WHERE key=$1 AND expires_at>$2`, key, now).Scan(&saved)
		if err == nil {
			if err = json.Unmarshal(saved, &link); err != nil { return domain.Link{}, err }
			return link, tx.Commit()
		}
		if !errors.Is(err, sql.ErrNoRows) { return domain.Link{}, err }
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO links(code,destination,created_at) VALUES($1,$2,$3) RETURNING id`, link.Code, link.URL, now).Scan(&link.ID)
	if err != nil { return domain.Link{}, err }
	var event map[string]json.RawMessage
	if err = json.Unmarshal([]byte(payload), &event); err != nil { return domain.Link{}, err }
	linkPayload, err := json.Marshal(link)
	if err != nil { return domain.Link{}, err }
	event["link"] = linkPayload
	payloadBytes, err := json.Marshal(event)
	if err != nil { return domain.Link{}, err }
	var envelope struct { EventID string `json:"event_id"` }
	if err = json.Unmarshal(payloadBytes, &envelope); err != nil { return domain.Link{}, err }
	if _, err = tx.ExecContext(ctx, `INSERT INTO outbox(event_id,topic,message_key,payload,created_at) VALUES($1,$2,$3,$4,$5)`, envelope.EventID, "linklet.events.v1", link.Code, string(payloadBytes), now); err != nil { return domain.Link{}, err }
	if key != "" {
		response, marshalErr := json.Marshal(link)
		if marshalErr != nil { return domain.Link{}, marshalErr }
		if _, err = tx.ExecContext(ctx, `INSERT INTO idempotency_keys(key,response,expires_at) VALUES($1,$2,$3) ON CONFLICT(key) DO UPDATE SET response=EXCLUDED.response,expires_at=EXCLUDED.expires_at`, key, string(response), now.Add(24*time.Hour)); err != nil { return domain.Link{}, err }
	}
	return link, tx.Commit()
}

func (s *Store) Get(ctx context.Context, code string) (domain.Link, error) {
	var link domain.Link
	err := s.DB.QueryRowContext(ctx, `SELECT id,code,destination,created_at FROM links WHERE code=$1`, code).Scan(&link.ID,&link.Code,&link.URL,&link.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) { return domain.Link{}, domain.ErrNotFound }
	return link, err
}

func (s *Store) List(ctx context.Context, before time.Time, id int64, limit int) ([]domain.Link, error) {
	var rows *sql.Rows
	var err error
	if id > 0 { rows, err = s.DB.QueryContext(ctx, `SELECT id,code,destination,created_at FROM links WHERE (created_at,id)<($1,$2) ORDER BY created_at DESC,id DESC LIMIT $3`, before,id,limit) } else { rows, err = s.DB.QueryContext(ctx, `SELECT id,code,destination,created_at FROM links ORDER BY created_at DESC,id DESC LIMIT $1`, limit) }
	if err != nil { return nil, err }
	defer rows.Close()
	items := make([]domain.Link,0,limit)
	for rows.Next() { var l domain.Link; if err := rows.Scan(&l.ID,&l.Code,&l.URL,&l.CreatedAt); err != nil { return nil,err }; items=append(items,l) }
	return items, rows.Err()
}

func (s *Store) Ready(ctx context.Context) error { return s.DB.PingContext(ctx) }

func (s *Store) OutboxBacklog(ctx context.Context) (int64, error) {
	var count int64
	err := s.DB.QueryRowContext(ctx, `SELECT count(*) FROM outbox WHERE published_at IS NULL`).Scan(&count)
	return count, err
}

func (s *Store) Pending(ctx context.Context, batch int) ([]OutboxEvent, error) {
	tx, err := s.DB.BeginTx(ctx,nil); if err != nil { return nil,err }; defer tx.Rollback()
	rows,err:=tx.QueryContext(ctx,`SELECT event_id,topic,message_key,payload FROM outbox WHERE published_at IS NULL ORDER BY created_at LIMIT $1 FOR UPDATE SKIP LOCKED`,batch)
	if err!=nil{return nil,err}; defer rows.Close()
	events:=make([]OutboxEvent,0,batch)
	for rows.Next(){var e OutboxEvent;if err:=rows.Scan(&e.ID,&e.Topic,&e.Key,&e.Payload);err!=nil{return nil,err};events=append(events,e)}
	if err=rows.Err();err!=nil{return nil,err}
	return events,tx.Commit()
}

type OutboxEvent struct { ID,Topic,Key string; Payload []byte }

func (s *Store) MarkPublished(ctx context.Context, id string) error {
	result,err:=s.DB.ExecContext(ctx,`UPDATE outbox SET published_at=now() WHERE event_id=$1 AND published_at IS NULL`,id)
	if err!=nil{return err};n,err:=result.RowsAffected();if err!=nil{return err};if n==0{return errors.New("outbox event was already marked")};return nil
}
