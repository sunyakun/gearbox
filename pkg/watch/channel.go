package watch

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
)

type channel[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	msgCh  <-chan *message.Message
}

func (ch *channel[T]) Stop() {
	ch.cancel()
}

func (ch *channel[T]) ResultChan() (<-chan Event[T], error) {
	var evtCh = make(chan Event[T])
	go func() {
		for {
			select {
			case msg := <-ch.msgCh:
				if msg == nil {
					return
				}
				var obj T
				if err := json.Unmarshal(msg.Payload, &obj); err != nil {
					evtCh <- Event[T]{Type: EventTypeError}
				}
				evtCh <- Event[T]{Type: EventType(msg.Metadata["Type"]), Obj: &obj}
				msg.Ack()
			case <-ch.ctx.Done():
				close(evtCh)
				return
			}
		}
	}()
	return evtCh, nil
}
