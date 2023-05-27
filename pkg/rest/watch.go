package rest

import (
	"context"

	"github.com/sunyakun/gearbox/pkg/apis"
	"github.com/sunyakun/gearbox/pkg/watch"
	"github.com/go-logr/logr"
)

type Channel interface {
	Stop()
	ResultChan() (<-chan Event, error)
}

type Event struct {
	Type watch.EventType
	Obj  apis.Object
}

type channel[T any, PT interface {
	apis.Object
	*T
}, ST any] struct {
	ctx       context.Context
	cancel    context.CancelFunc
	channel   watch.Channel[ST]
	logger    logr.Logger
	converter Converter[PT, ST]
	scheme    *apis.Scheme
}

func NewChannel[T any, PT interface {
	apis.Object
	*T
}, ST any](storeChannel watch.Channel[ST], scheme *apis.Scheme, logger logr.Logger, converter Converter[PT, ST]) Channel {
	ctx, cancel := context.WithCancel(context.Background())
	return &channel[T, PT, ST]{
		ctx:       ctx,
		cancel:    cancel,
		channel:   storeChannel,
		logger:    logger,
		converter: converter,
		scheme:    scheme,
	}
}

func (c *channel[T, PT, ST]) Stop() {
	c.channel.Stop()
}

func (c *channel[T, PT, ST]) ResultChan() (<-chan Event, error) {
	ch := make(chan Event)
	resultCh, err := c.channel.ResultChan()
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			select {
			case evt := <-resultCh:
				origObj, ok := interface{}(evt.Obj).(*ST)
				if !ok {
					c.logger.Error(nil, "receive an unexpected object from the storage channel", "event", evt)
					continue
				}
				var obj = PT(new(T))
				if err := c.converter.FromStorage(origObj, obj); err != nil {
					c.logger.Error(err, "convert storage object to api object failed", "storageObject", origObj)
					continue
				}
				kind, err := c.scheme.ObjectKind(obj)
				if err != nil {
					c.logger.Error(err, "failed get object kind", "object", obj)
				}
				obj.SetKind(kind)
				ch <- Event{Type: evt.Type, Obj: obj}
			case <-c.ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}
