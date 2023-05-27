package rest

import (
	"context"

	"github.com/sunyakun/gearbox/pkg/admission"
	"github.com/sunyakun/gearbox/pkg/apis"
	"github.com/sunyakun/gearbox/pkg/errors"
	"github.com/sunyakun/gearbox/pkg/storage"
	"github.com/sunyakun/gearbox/pkg/storage/selector"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-logr/logr"
)

var _ WatchableClient[*apis.ObjectMeta] = &RestAPI[apis.ObjectMeta, *apis.ObjectMeta, any]{}

type RestAPI[T any, PT interface {
	apis.Object
	*T
}, ST any] struct {
	converter    Converter[PT, ST]
	store        storage.WatchableStore[ST]
	resourceName string
	version      string
	logger       logr.Logger
	admit        admission.Interface
	scheme       *apis.Scheme
}

func NewRestAPI[T any, PT interface {
	apis.Object
	*T
}, ST any](
	resourceName string, store storage.WatchableStore[ST], scheme *apis.Scheme, converter Converter[PT, ST], logger logr.Logger, admits []admission.Interface,
) *RestAPI[T, PT, ST] {

	return &RestAPI[T, PT, ST]{
		converter:    converter,
		store:        store,
		resourceName: resourceName,
		version:      "v1",
		logger:       logger,
		admit:        admission.NewChainHandler(admits...),
		scheme:       scheme,
	}
}

func (rest *RestAPI[T, PT, ST]) Name() string {
	return rest.resourceName
}

func (rest *RestAPI[T, PT, ST]) Version() string {
	return rest.version
}

func (rest *RestAPI[T, PT, ST]) convertStorageError(err error, obj apis.Object) error {
	kind, e := rest.scheme.ObjectKind(obj)
	if e != nil {
		return e
	}
	switch {
	case storage.IsNotFoundError(err):
		return errors.NewNotFound(kind, obj.GetKey())
	case storage.IsAlreadyExistError(err):
		return errors.NewConflict(err)
	case storage.IsConcurrentConclictError(err):
		return errors.NewConflict(err)
	}
	return err
}

func (rest *RestAPI[T, PT, ST]) doAdmit(ctx context.Context, operation admission.Operation, obj apis.Object) error {
	attrs := &admission.Attribute{
		Object:       obj,
		Operation:    operation,
		ResourceName: rest.resourceName,
	}
	if rest.admit != admission.Interface(nil) && rest.admit.Handles(operation) {
		validation, ok := rest.admit.(admission.ValidationInterface)
		if ok {
			if err := validation.Validate(ctx, attrs); err != nil {
				rest.logger.Error(err, "do validator admit failed", "operation", operation)
				return err
			}
		}
		mutation, ok := rest.admit.(admission.MutationInterface)
		if ok {
			if err := mutation.Admit(ctx, attrs); err != nil {
				rest.logger.Error(err, "do mutation admission failed", "operation", operation)
				return err
			}
		}
	}
	return nil
}

func (rest *RestAPI[T, PT, ST]) Get(ctx context.Context, key string) (PT, error) {
	var obj = PT(new(T))
	obj.SetKey(key)
	storeObj, err := rest.store.Get(ctx, key)
	if err != nil {
		return nil, rest.convertStorageError(err, obj)
	}
	if err := rest.converter.FromStorage(storeObj, obj); err != nil {
		return nil, err
	}
	kind, err := rest.scheme.ObjectKind(obj)
	if err != nil {
		return nil, err
	}
	obj.SetKind(kind)
	return obj, nil
}

func (rest *RestAPI[T, PT, ST]) GetList(ctx context.Context, opts apis.ListOptions) ([]PT, int64, error) {
	if opts.Offset <= 0 {
		opts.Offset = 0
	}
	if opts.Limit < 0 {
		opts.Limit = 0
	}

	requirements, err := selector.Parse(opts.Selector)
	if err != nil {
		return nil, 0, err
	}

	storeObjs, count, err := rest.store.GetList(ctx, storage.ListOptions{
		Offset:       opts.Offset,
		Limit:        opts.Limit,
		Requirements: requirements,
	})
	if err != nil {
		return nil, 0, rest.convertStorageError(err, PT(new(T)))
	}
	var outs []PT
	for _, storeobj := range storeObjs {
		var obj = PT(new(T))
		if err := rest.converter.FromStorage(storeobj, obj); err != nil {
			return nil, 0, err
		}
		kind, err := rest.scheme.ObjectKind(obj)
		if err != nil {
			return nil, 0, err
		}
		obj.SetKind(kind)
		outs = append(outs, obj)
	}
	return outs, count, err
}

func (rest *RestAPI[T, PT, ST]) Create(ctx context.Context, obj PT) (PT, error) {
	if obj.GetKey() == "" {
		return nil, errors.NewBadRequest("the key can't be empty")
	}
	if err := rest.doAdmit(ctx, admission.Create, obj); err != nil {
		return nil, err
	}
	var storeObj = new(ST)
	if err := rest.converter.ToStorage(obj, storeObj); err != nil {
		return nil, err
	}
	newStoreObj, err := rest.store.Create(ctx, storeObj)
	if err != nil {
		return nil, rest.convertStorageError(err, obj)
	}
	if err := rest.converter.FromStorage(newStoreObj, obj); err != nil {
		return nil, err
	}
	kind, err := rest.scheme.ObjectKind(obj)
	if err != nil {
		return nil, err
	}
	obj.SetKind(kind)
	return obj, nil
}

func (rest *RestAPI[T, PT, ST]) Update(ctx context.Context, key string, obj PT) error {
	obj.SetKey(key)
	if err := rest.doAdmit(ctx, admission.Update, obj); err != nil {
		return err
	}
	var storeObj = new(ST)
	if err := rest.converter.ToStorage(obj, storeObj); err != nil {
		return err
	}
	if err := rest.store.Update(ctx, key, storeObj); err != nil {
		return rest.convertStorageError(err, obj)
	}
	if err := rest.converter.FromStorage(storeObj, obj); err != nil {
		return err
	}
	return nil
}

func (rest *RestAPI[T, PT, ST]) Delete(ctx context.Context, key string) error {
	var obj = PT(new(T))
	obj.SetKey(key)
	if err := rest.doAdmit(ctx, admission.Delete, &apis.ObjectMeta{
		Key: key,
	}); err != nil {
		return err
	}
	return rest.convertStorageError(rest.store.Delete(ctx, key, nil), obj)
}

func (rest *RestAPI[T, PT, ST]) Watch(ctx context.Context) (Channel, error) {
	channel, err := rest.store.Watch(ctx)
	if err != nil {
		return nil, err
	}
	return NewChannel(channel, rest.scheme, rest.logger, rest.converter), nil
}

func (rest *RestAPI[T, PT, ST]) Install(container *restful.Container) {
	handler := NewHandler[T, PT](rest)
	handler.AddToContainer(container)
}
