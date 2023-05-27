package controller

import (
	"context"

	"github.com/sunyakun/gearbox/pkg/reconcile"
)

// EventHandler enqueues reconcile.Requests in response to events (e.g. Pod Create).  EventHandlers map an Event
// for one object to trigger Reconciles for either the same object or different objects - e.g. if there is an
// Event for object with type Foo (using source.KindSource) then reconcile one or more object(s) with type Bar.
//
// Identical reconcile.Requests will be batched together through the queuing mechanism before reconcile is called.
//
// * Use EnqueueRequestForObject to reconcile the object the event is for
// - do this for events for the type the Controller Reconciles. (e.g. Deployment for a Deployment Controller)
//
// * Use EnqueueRequestForOwner to reconcile the owner of the object the event is for
// - do this for events for the types the Controller creates.  (e.g. ReplicaSets created by a Deployment Controller)
//
// * Use EnqueueRequestsFromMapFunc to transform an event for an object to a reconcile of an object
// of a different type - do this for events for types the Controller may be interested in, but doesn't create.
// (e.g. If Foo responds to cluster size events, map Node events to Foo objects.)
//
// Unless you are implementing your own EventHandler, you can ignore the functions on the EventHandler interface.
// Most users shouldn't need to implement their own EventHandler.
type EventHandler interface {
	// Create is called in response to an create event - e.g. Pod Creation.
	Create(context.Context, CreateEvent, RateLimiter)

	// Update is called in response to an update event -  e.g. Pod Updated.
	Update(context.Context, UpdateEvent, RateLimiter)

	// Delete is called in response to a delete event - e.g. Pod Deleted.
	Delete(context.Context, DeleteEvent, RateLimiter)

	// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
	// external trigger request - e.g. reconcile Autoscaling, or a Webhook.
	Generic(context.Context, GenericEvent, RateLimiter)
}

var _ EventHandler = Funcs{}

// Funcs implements EventHandler.
type Funcs struct {
	// Create is called in response to an add event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.Requests.
	CreateFunc func(context.Context, CreateEvent, RateLimiter)

	// Update is called in response to an update event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.Requests.
	UpdateFunc func(context.Context, UpdateEvent, RateLimiter)

	// Delete is called in response to a delete event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.Requests.
	DeleteFunc func(context.Context, DeleteEvent, RateLimiter)

	// GenericFunc is called in response to a generic event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.Requests.
	GenericFunc func(context.Context, GenericEvent, RateLimiter)
}

// Create implements EventHandler.
func (h Funcs) Create(ctx context.Context, e CreateEvent, q RateLimiter) {
	if h.CreateFunc != nil {
		h.CreateFunc(ctx, e, q)
	}
}

// Delete implements EventHandler.
func (h Funcs) Delete(ctx context.Context, e DeleteEvent, q RateLimiter) {
	if h.DeleteFunc != nil {
		h.DeleteFunc(ctx, e, q)
	}
}

// Update implements EventHandler.
func (h Funcs) Update(ctx context.Context, e UpdateEvent, q RateLimiter) {
	if h.UpdateFunc != nil {
		h.UpdateFunc(ctx, e, q)
	}
}

// Generic implements EventHandler.
func (h Funcs) Generic(ctx context.Context, e GenericEvent, q RateLimiter) {
	if h.GenericFunc != nil {
		h.GenericFunc(ctx, e, q)
	}
}

var EnqueueHandler = Funcs{
	CreateFunc: func(ctx context.Context, evt CreateEvent, q RateLimiter) {
		if evt.Object != nil {
			q.Add(reconcile.Request{
				Key: evt.Object.GetKey(),
			})
		}
	},
	UpdateFunc: func(ctx context.Context, evt UpdateEvent, q RateLimiter) {
		if evt.ObjectNew != nil {
			q.Add(reconcile.Request{
				Key: evt.ObjectNew.GetKey(),
			})
		}
	},
	DeleteFunc: func(ctx context.Context, evt DeleteEvent, q RateLimiter) {
		if evt.Object != nil {
			q.Add(reconcile.Request{
				Key: evt.Object.GetKey(),
			})
		}
	},
	GenericFunc: func(ctx context.Context, evt GenericEvent, q RateLimiter) {
		if evt.Object != nil {
			q.Add(reconcile.Request{
				Key: evt.Object.GetKey(),
			})
		}
	},
}
