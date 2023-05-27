package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/sunyakun/gearbox/pkg/reconcile"
	"k8s.io/client-go/util/workqueue"
)

type WatchDescribe struct {
	name       string
	src        Source
	handler    EventHandler
	predicates []Predicate
}

func (wd *WatchDescribe) Name() string {
	return wd.name
}

func (wd *WatchDescribe) Start(ctx context.Context, rateLimiter RateLimiter) error {
	return wd.src.Start(ctx, wd.handler, rateLimiter, wd.predicates...)
}

func NewWatchDescribe(name string, src Source, handler EventHandler, predicates ...Predicate) *WatchDescribe {
	return &WatchDescribe{
		name:       name,
		src:        src,
		handler:    handler,
		predicates: predicates,
	}
}

type RateLimiter = workqueue.RateLimitingInterface

type WatchDescribeInterface interface {
	Name() string
	Start(context.Context, RateLimiter) error
}

type Controller interface {
	reconcile.Reconciler

	// Watch takes events provided by a Source and uses the EventHandler to
	// enqueue reconcile.Requests in response to the events.
	//
	// Watch may be provided one or more Predicates to filter events before
	// they are given to the EventHandler.  Events will be passed to the
	// EventHandler if all provided Predicates evaluate to true.
	Watch(WatchDescribeInterface) error

	// Start starts the controller.  Start blocks until the context is closed or a
	// controller has an error starting.
	Start(ctx context.Context) error
}

type ControllerConfig struct {
	MaxConcurrentReconciles int
	RecoverPanic            bool
	Reconciler              reconcile.Reconciler
	Logger                  logr.Logger
}

type controller struct {
	ctx            context.Context
	watchDescribes []WatchDescribeInterface

	Name                    string
	Logger                  logr.Logger
	Do                      reconcile.Reconciler
	MaxConcurrentReconciles int
	RecoverPanic            *bool

	Started bool
	Queue   RateLimiter
}

func New(name string, config ControllerConfig) Controller {
	if config.Logger.GetSink() == nil {
		config.Logger = logrusr.New(logrus.New())
	}

	ctl := &controller{
		Name:                    name,
		MaxConcurrentReconciles: config.MaxConcurrentReconciles,
		RecoverPanic:            lo.ToPtr(config.RecoverPanic),
		Logger:                  config.Logger,
		Do:                      config.Reconciler,
	}

	return ctl
}

func (ctl *controller) Watch(describe WatchDescribeInterface) error {
	if !ctl.Started {
		ctl.watchDescribes = append(ctl.watchDescribes, describe)
		return nil
	}
	if err := describe.Start(ctl.ctx, ctl.Queue); err != nil {
		return err
	}
	return nil
}

func (ctl *controller) Start(ctx context.Context) error {
	if ctl.Started {
		return fmt.Errorf("the controller '%s' has already started", ctl.Name)
	}

	rateLimiter := workqueue.DefaultControllerRateLimiter()
	ctl.Queue = workqueue.NewNamedRateLimitingQueue(rateLimiter, ctl.Name)
	ctl.ctx = ctx

	go func() {
		<-ctx.Done()
		ctl.Queue.ShutDown()
	}()

	for _, desc := range ctl.watchDescribes {
		ctl.Logger.Info("start watch", "controller", ctl.Name, "resource", desc.Name())
		err := desc.Start(ctx, ctl.Queue)
		if err != nil {
			return err
		}
	}
	ctl.watchDescribes = nil
	ctl.Started = true

	wg := sync.WaitGroup{}
	wg.Add(ctl.MaxConcurrentReconciles)
	for i := 0; i < ctl.MaxConcurrentReconciles; i++ {
		go func() {
			defer wg.Done()
			for ctl.processNextWorkItem(ctx) {
			}
		}()
	}

	ctl.Logger.Info("the controller started", "name", ctl.Name, "MaxConcurrentReconciles", ctl.MaxConcurrentReconciles)

	<-ctx.Done()
	wg.Wait()
	return nil
}

// Reconcile implements reconcile.Reconciler.
func (ctl *controller) Reconcile(ctx context.Context, req reconcile.Request) (_ reconcile.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			if ctl.RecoverPanic != nil && *ctl.RecoverPanic {
				err = fmt.Errorf("panic: %v [recovered]", r)
				ctl.Logger.Error(err, "observed a panic in reconciler")
				return
			}
			ctl.Logger.Error(nil, fmt.Sprintf("Observed a panic in reconciler: %v", r))
			panic(r)
		}
	}()
	return ctl.Do.Reconcile(ctx, req)
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the reconcileHandler.
func (ctl *controller) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := ctl.Queue.Get()
	if shutdown {
		// Stop working
		return false
	}

	// We call Done here so the workqueue knows we have finished
	// processing this item. We also must remember to call Forget if we
	// do not want this work item being re-queued. For example, we do
	// not call Forget if a transient error occurs, instead the item is
	// put back on the workqueue and attempted again after a back-off
	// period.
	defer ctl.Queue.Done(obj)

	ctl.reconcileHandler(ctx, obj)
	return true
}

func (ctl *controller) reconcileHandler(ctx context.Context, obj interface{}) {
	// Make sure that the object is a valid request.
	req, ok := obj.(reconcile.Request)
	if !ok {
		// As the item in the workqueue is actually invalid, we call
		// Forget here else we'd go into a loop of attempting to
		// process a work item that is invalid.
		ctl.Queue.Forget(obj)
		// Return true, don't take a break
		return
	}

	// resource to be synced.
	result, err := ctl.Reconcile(ctx, req)
	switch {
	case err != nil:
		ctl.Queue.AddRateLimited(req)
		ctl.Logger.Error(err, "Reconciler error")
	case result.RequeueAfter > 0:
		// The result.RequeueAfter request will be lost, if it is returned
		// along with a non-nil error. But this is intended as
		// We need to drive to stable reconcile loops before queuing due
		// to result.RequestAfter
		ctl.Queue.Forget(obj)
		ctl.Queue.AddAfter(req, result.RequeueAfter)
	case result.Requeue:
		ctl.Queue.AddRateLimited(req)
	default:
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		ctl.Queue.Forget(obj)
	}
}
