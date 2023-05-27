package storage

import (
	"context"

	"github.com/sunyakun/gearbox/pkg/storage/selector"
	"github.com/sunyakun/gearbox/pkg/watch"
)

type ListOptions struct {
	Offset       int
	Limit        int
	Requirements []selector.Requirement
}

type Store[T any] interface {
	Get(ctx context.Context, key string) (*T, error)

	GetList(ctx context.Context, opts ListOptions) ([]*T, int64, error)

	Create(ctx context.Context, obj *T) (*T, error)

	Update(ctx context.Context, key string, obj *T) error

	// Delete remove the object specified by key. If the key don't exists, it will
	// return NotFound error
	Delete(ctx context.Context, key string, obj *T) error
}

type WatchableStore[T any] interface {
	watch.Watcher[T]
	Store[T]
}
