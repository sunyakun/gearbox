package storage

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"reflect"

// 	"github.com/ThreeDotsLabs/watermill"
// 	"github.com/ThreeDotsLabs/watermill/message"
// )

// type EventType string

// var (
// 	EventTypeCreated EventType = "Created"
// 	EventTypeUpdated EventType = "Updated"
// 	EventTypeDeleted EventType = "Deleted"
// 	EventTypeError   EventType = "Error"
// )

// type PubSub interface {
// 	message.Publisher
// 	message.Subscriber
// }

// type EventPublisher[T any] interface {
// 	Publish(ctx context.Context, eventType EventType, obj *T) error
// 	Close() error
// }

// type Watcher[T any] interface {
// 	Watch(context.Context) (Channel[T], error)
// }

// type EventPubWatcher[T any] interface {
// 	EventPublisher[T]
// 	Watcher[T]
// }

// type Channel[T any] interface {
// 	Stop()
// 	ResultChan() (<-chan Event[T], error)
// }

// type Event[T any] struct {
// 	Type EventType
// 	Obj  *T
// }

// type channel[T any] struct {
// 	ctx    context.Context
// 	cancel context.CancelFunc
// 	msgCh  <-chan *message.Message
// }

// func (ch *channel[T]) Stop() {
// 	ch.cancel()
// }

// func (ch *channel[T]) ResultChan() (<-chan Event[T], error) {
// 	var evtCh = make(chan Event[T])
// 	go func() {
// 		for {
// 			select {
// 			case msg := <-ch.msgCh:
// 				if msg == nil {
// 					return
// 				}
// 				var obj T
// 				if err := json.Unmarshal(msg.Payload, &obj); err != nil {
// 					evtCh <- Event[T]{Type: EventTypeError}
// 				}
// 				evtCh <- Event[T]{Type: EventType(msg.Metadata["Type"]), Obj: &obj}
// 				msg.Ack()
// 			case <-ch.ctx.Done():
// 				close(evtCh)
// 				return
// 			}
// 		}
// 	}()
// 	return evtCh, nil
// }

// type pubwatcher[T any] struct {
// 	topic  string
// 	pubsub PubSub
// }

// func NewPubWatcher[T any](wmPubsub PubSub) (EventPubWatcher[T], error) {
// 	p := &pubwatcher[T]{pubsub: wmPubsub}
// 	p.topic = genTopicName[T]()
// 	if p.topic == "" {
// 		return nil, fmt.Errorf("the generic type T must have a name")
// 	}
// 	return p, nil
// }

// func (p *pubwatcher[T]) Close() error {
// 	return p.pubsub.Close()
// }

// func (p *pubwatcher[T]) Publish(ctx context.Context, eventType EventType, obj *T) error {
// 	payload, err := json.Marshal(obj)
// 	if err != nil {
// 		return err
// 	}
// 	msg := message.NewMessage(watermill.NewShortUUID(), payload)
// 	msg.Metadata["Type"] = string(eventType)
// 	return p.pubsub.Publish(p.topic, msg)
// }

// func (p *pubwatcher[T]) Watch(ctx context.Context) (Channel[T], error) {
// 	msgCh, err := p.pubsub.Subscribe(ctx, p.topic)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ctx, cancel := context.WithCancel(context.Background())
// 	return &channel[T]{msgCh: msgCh, ctx: ctx, cancel: cancel}, nil
// }

// func genTopicName[T any]() string {
// 	var t T
// 	rt := reflect.TypeOf(t)
// 	if rt.PkgPath() == "" || rt.Name() == "" {
// 		return ""
// 	}
// 	return fmt.Sprintf("%s:%s", rt.PkgPath(), rt.Name())
// }
