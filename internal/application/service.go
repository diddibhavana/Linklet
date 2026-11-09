package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"linklet/internal/domain"
	"linklet/internal/ports"
)

var ErrNotFound = errors.New("link not found")

type Service struct {
	store ports.Store
	cache ports.Cache
	clock func() time.Time
	code  func() (string, error)
}

func New(store ports.Store, cache ports.Cache) *Service {
	return &Service{store: store, cache: cache, clock: time.Now, code: randomCode}
}

func randomCode() (string, error) {
	b := make([]byte, 9)
	if _, err := rand.Read(b); err != nil { return "", err }
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) Create(ctx context.Context, rawURL, custom, idempotencyKey string) (domain.Link, error) {
	destination, err := domain.ValidateDestination(rawURL)
	if err != nil { return domain.Link{}, err }
	code := custom
	if code != "" && !domain.ValidateCode(code) { return domain.Link{}, domain.ErrInvalidCode }
	if code == "" {
		code, err = s.code()
		if err != nil { return domain.Link{}, fmt.Errorf("generate code: %w", err) }
	}
	now := s.clock().UTC()
	link := domain.Link{Code: code, URL: destination, CreatedAt: now}
	eventID, err := randomCode()
	if err != nil { return domain.Link{}, err }
	payload, err := json.Marshal(map[string]any{"event_id": eventID, "version": 1, "type": "link.created", "occurred_at": now, "link": link})
	if err != nil { return domain.Link{}, err }
	return s.store.Create(ctx, link, idempotencyKey, string(payload), now)
}

func (s *Service) Resolve(ctx context.Context, code string) (domain.Link, error) {
	if value, err := s.cache.Get(ctx, code); err == nil && value != "" {
		return domain.Link{Code: code, URL: value}, nil
	}
	link, err := s.store.Get(ctx, code)
	if err != nil { return domain.Link{}, err }
	_ = s.cache.Set(ctx, code, link.URL, 6*time.Hour)
	return link, nil
}

func (s *Service) List(ctx context.Context, before time.Time, id int64, limit int) ([]domain.Link, error) {
	return s.store.List(ctx, before, id, limit)
}
