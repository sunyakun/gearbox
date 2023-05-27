package watch

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type EventType string

var (
	EventTypeCreated EventType = "Created"
	EventTypeUpdated EventType = "Updated"
	EventTypeDeleted EventType = "Deleted"
	EventTypeGeneric EventType = "Generic"
	EventTypeError   EventType = "Error"
)

type PubSub interface {
	message.Publisher
	message.Subscriber
}

type EventPublisher[T any] interface {
	Publish(ctx context.Context, eventType EventType, obj *T) error
	Close() error
}

type Watcher[T any] interface {
	Watch(context.Context) (Channel[T], error)
}

type EventPubWatcher[T any] interface {
	EventPublisher[T]
	Watcher[T]
}

type Channel[T any] interface {
	Stop()
	ResultChan() (<-chan Event[T], error)
}

type Event[T any] struct {
	Type EventType
	Obj  *T
}
