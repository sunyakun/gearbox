package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

var (
	once            sync.Once
	watermillPubSub PubSub
)

type pubwatcher[T any] struct {
	topic  string
	pubsub PubSub
}

func NewPubSub[T any]() (EventPubWatcher[T], error) {
	once.Do(func() {
		watermillPubSub = gochannel.NewGoChannel(gochannel.Config{}, &watermill.NopLogger{})
	})
	p := &pubwatcher[T]{pubsub: watermillPubSub}
	p.topic = genTopicName[T]()
	if p.topic == "" {
		return nil, fmt.Errorf("the generic type T must have a name")
	}
	return p, nil
}

func (p *pubwatcher[T]) Close() error {
	return p.pubsub.Close()
}

func (p *pubwatcher[T]) Publish(ctx context.Context, eventType EventType, obj *T) error {
	payload, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	msg := message.NewMessage(watermill.NewShortUUID(), payload)
	msg.Metadata["Type"] = string(eventType)
	return p.pubsub.Publish(p.topic, msg)
}

func (p *pubwatcher[T]) Watch(ctx context.Context) (Channel[T], error) {
	msgCh, err := p.pubsub.Subscribe(ctx, p.topic)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	return &channel[T]{msgCh: msgCh, ctx: ctx, cancel: cancel}, nil
}

func genTopicName[T any]() string {
	var t T
	rt := reflect.TypeOf(t)
	if rt.PkgPath() == "" || rt.Name() == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s", rt.PkgPath(), rt.Name())
}
