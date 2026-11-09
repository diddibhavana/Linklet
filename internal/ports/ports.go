package ports

import (
	"context"
	"time"

	"linklet/internal/domain"
)

type Store interface {
	Create(context.Context, domain.Link, string, string, time.Time) (domain.Link, error)
	Get(context.Context, string) (domain.Link, error)
	List(context.Context, time.Time, int64, int) ([]domain.Link, error)
	Ready(context.Context) error
}

type Cache interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string, time.Duration) error
	Ready(context.Context) error
}

type Events interface {
	Publish(context.Context, string, []byte) error
}
