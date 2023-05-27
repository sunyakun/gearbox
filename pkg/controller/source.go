package controller

import (
	"context"
	"time"

	"github.com/sunyakun/gearbox/pkg/rest"
	"github.com/sunyakun/gearbox/pkg/watch"
)

type Source interface {
	// Start is internal and should be called only by the Controller to register an EventHandler with the Informer
	// to enqueue reconcile.Requests.
	Start(context.Context, EventHandler, RateLimiter, ...Predicate) error
}

type source struct {
	channel rest.Channel
}

// NewSource create a Srouce to handle the storage Create/Update/Delete event
func NewSource(channel rest.Channel) *source {
	return &source{
		channel: channel,
	}
}

func (s *source) Start(ctx context.Context, eventHandler EventHandler, rateLimiter RateLimiter, predicates ...Predicate) error {
	eventCh, err := s.channel.ResultChan()
	if err != nil {
		return err
	}
	go func() {
	MAIN_LOOP:
		for {
			select {
			case evt := <-eventCh:
				if evt.Obj.GetKey() == "" {
					continue
				}
				switch evt.Type {
				case watch.EventTypeCreated:
					createEvent := CreateEvent{Object: evt.Obj}
					for _, predicate := range predicates {
						if !predicate.Create(createEvent) {
							continue MAIN_LOOP
						}
					}
					eventHandler.Create(ctx, createEvent, rateLimiter)
				case watch.EventTypeUpdated:
					updateEvent := UpdateEvent{ObjectNew: evt.Obj}
					for _, predicate := range predicates {
						if !predicate.Update(updateEvent) {
							continue MAIN_LOOP
						}
					}
					eventHandler.Update(ctx, updateEvent, rateLimiter)
				case watch.EventTypeDeleted:
					deleteEvent := DeleteEvent{Object: evt.Obj}
					for _, predicate := range predicates {
						if !predicate.Delete(deleteEvent) {
							continue MAIN_LOOP
						}
					}
					eventHandler.Delete(ctx, deleteEvent, rateLimiter)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

type TimerSource struct {
	tick *time.Ticker
}

func NewTimerSource(duration time.Duration) Source {
	return &TimerSource{
		tick: time.NewTicker(duration),
	}
}

func (s *TimerSource) Start(ctx context.Context, eventHandler EventHandler, rateLimiter RateLimiter, predicates ...Predicate) error {
	go func() {
		for {
			select {
			case <-s.tick.C:
				evt := GenericEvent{}
				for _, predicate := range predicates {
					if !predicate.Generic(evt) {
						continue
					}
				}
				eventHandler.Generic(ctx, evt, rateLimiter)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}
