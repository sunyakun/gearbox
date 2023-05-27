package rest

import (
	"context"

	"github.com/sunyakun/gearbox/pkg/apis"
	"github.com/emicklei/go-restful/v3"
)

// Converter knowns how to do convert between api type and storage type
type Converter[T apis.Object, ST any] interface {
	FromStorage(from *ST, to T) error
	ToStorage(from T, to *ST) error
}

type Client[T apis.Object] interface {
	Get(ctx context.Context, key string) (T, error)
	GetList(ctx context.Context, opts apis.ListOptions) ([]T, int64, error)
	Create(ctx context.Context, obj T) (T, error)
	Update(ctx context.Context, key string, obj T) error
	Delete(ctx context.Context, key string) error
}

type WatchableClient[T apis.Object] interface {
	Client[T]
	Watch(ctx context.Context) (Channel, error)
}

type Resource[T apis.Object] interface {
	WatchableClient[T]
	Name() string
	Version() string
	Install(*restful.Container)
}
