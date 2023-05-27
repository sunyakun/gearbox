package gorm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"

	"github.com/sunyakun/gearbox/pkg/storage"
	"github.com/sunyakun/gearbox/pkg/util"
	"github.com/sunyakun/gearbox/pkg/watch"
)

var _ storage.WatchableStore[any] = &store[any, any]{}

// Config used to construct store.
// <keyFieldName> should be the unique key used to select the object from the underlying SQL database.
// <RevisionFieldName> is used to implement the optimistic lock to avoid data races.
// If this field is empty, concurrent update and delete operations will be unsafe.
// <FieldGetter> can be obtained from the gorm/gen generated code.
// <ParseToTime> is used to convert the string-formatted time to time.Time{}.
type Config struct {
	KeyColumnName      string
	RevisionColumnName string
	FieldGetter        FieldGetter
	ParseToTime        func(string) (time.Time, error)
}

type store[GormModelT, GenDoT any] struct {
	db             *gorm.DB
	typeName       string
	genDaoGetter   func(context.Context) GenDoT
	columns        []string
	keyFieldName   string
	keyFieldOffset uintptr
	rvFieldName    string
	rvFieldOffset  uintptr
	pubwatcher     watch.EventPubWatcher[GormModelT]
	selector       *Selector
	fieldGetter    FieldGetter
	onUpdate       []func(oldObj *GormModelT, newObj *GormModelT)
	onCreate       []func(*GormModelT)
}

// New create gorm/gen based store that implement the storage.Store interface.
// <daoGetter> will be used to get the gorm DO instance.
func New[GormModelT, GenDoT any](db *gorm.DB, daoGetter func(context.Context) GenDoT, cfg Config) (*store[GormModelT, GenDoT], error) {
	// make sure the `GormModelT` is a go struct
	gormModelRt, err := util.ReflectDefinedStruct[GormModelT]()
	if err != nil {
		return nil, err
	}

	pubwatcher, err := watch.NewPubSub[GormModelT]()
	if err != nil {
		return nil, err
	}

	keyField, ok := util.GetFieldByGormColumnTag(gormModelRt, cfg.KeyColumnName)
	if !ok {
		return nil, fmt.Errorf("type %s.%s have no field named '%s'", gormModelRt.PkgPath(), gormModelRt.Name(), cfg.KeyColumnName)
	}
	if keyField.Type.Kind() != reflect.String {
		return nil, fmt.Errorf("%s.%s must be string", gormModelRt.Name(), cfg.KeyColumnName)
	}

	s := &store[GormModelT, GenDoT]{
		db:             db,
		typeName:       gormModelRt.Name(),
		genDaoGetter:   daoGetter,
		columns:        util.InspectColumns(gormModelRt),
		keyFieldName:   cfg.KeyColumnName,
		rvFieldName:    cfg.RevisionColumnName,
		pubwatcher:     pubwatcher,
		selector:       NewSelector(cfg.FieldGetter, cfg.ParseToTime),
		keyFieldOffset: keyField.Offset,
		fieldGetter:    cfg.FieldGetter,
	}

	if cfg.RevisionColumnName != "" {
		revisionField, ok := util.GetFieldByGormColumnTag(gormModelRt, cfg.RevisionColumnName)
		if !ok {
			return nil, fmt.Errorf("type %s have no field named '%s'", gormModelRt.Name(), cfg.RevisionColumnName)
		}
		if revisionField.Type.Kind() != reflect.String {
			return nil, fmt.Errorf("%s.%s must be string", gormModelRt.Name(), cfg.RevisionColumnName)
		}
		s.rvFieldOffset = revisionField.Offset
	}

	return s, nil
}

func (s *store[GormModelT, GenDoT]) AddOnUpdateHandler(handler func(*GormModelT, *GormModelT)) {
	s.onUpdate = append(s.onUpdate, handler)
}

func (s *store[GormModelT, GenDoT]) AddOnCreateHandler(handler func(*GormModelT)) {
	s.onCreate = append(s.onCreate, handler)
}

func (s *store[GormModelT, GenDoT]) Get(ctx context.Context, key string) (out *GormModelT, err error) {
	err = s.db.Transaction(func(tx *gorm.DB) error {
		dao, err := NewDao[GormModelT](s.genDaoGetter(ctx), s.fieldGetter)
		if err != nil {
			return err
		}

		obj, err := dao.WithEqual(s.keyFieldName, key).First()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return storage.NewNotFoundError(s.typeName, key)
			}
			return err
		}
		out = obj
		return nil
	})
	return
}

func (s *store[GormModelT, GenDoT]) GetList(ctx context.Context, opts storage.ListOptions) ([]*GormModelT, int64, error) {
	conditions, err := s.selector.GenerateConditions(opts.Requirements)
	if err != nil {
		return nil, 0, err
	}
	dao, err := NewDao[GormModelT](s.genDaoGetter(ctx), s.fieldGetter)
	if err != nil {
		return nil, 0, err
	}
	return dao.Where(conditions...).FindByPage(opts.Offset, opts.Limit)
}

func (s *store[GormModelT, GenDoT]) Create(ctx context.Context, obj *GormModelT) (out *GormModelT, err error) {
	for _, handler := range s.onCreate {
		handler(obj)
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		dao, err := NewDao[GormModelT](s.genDaoGetter(ctx), s.fieldGetter)
		if err != nil {
			return err
		}

		if s.rvFieldName != "" {
			util.SetStringField(obj, s.rvFieldOffset, "1")
		}

		if err := dao.Create(obj); err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				return storage.NewAlreadyExistError(s.typeName, util.GetStringField(obj, s.keyFieldOffset))
			}
			return err
		}
		out = obj
		return s.pubwatcher.Publish(ctx, watch.EventTypeCreated, out)
	})
	return
}

func (s *store[GormModelT, GenDoT]) modify(ctx context.Context, dao *Dao[GormModelT, GenDoT], key string, obj *GormModelT, opFn func(dao *Dao[GormModelT, GenDoT], obj *GormModelT) (gen.ResultInfo, error)) (err error) {
	var (
		origDao = dao.WithEqual(s.keyFieldName, key)
	)

	dao = origDao

	var (
		rvInReq string
		result  gen.ResultInfo
	)
	if s.rvFieldName != "" {
		// modify with revision
		rvInReq = util.GetStringField(obj, s.rvFieldOffset)
		if rvInReq != "" {
			dao = dao.WithEqual(s.rvFieldName, rvInReq)
		}
	}

	result, err = opFn(dao, obj)
	if err != nil {
		return err
	}

	if result.RowsAffected != 1 {
		oldObj, err := origDao.First()
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			return storage.NewNotFoundError(s.typeName, key)
		} else if err != nil {
			return err
		}

		if rvInReq != "" {
			oldRv := util.GetStringField(oldObj, s.rvFieldOffset)
			if oldRv != rvInReq {
				return storage.NewConcurrentConclictError()
			}
		}
	}

	return nil
}

func (s *store[GormModelT, GenDoT]) Update(ctx context.Context, key string, obj *GormModelT) (err error) {
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var resourceVersion = "0"
		dao, err := NewDao[GormModelT](s.genDaoGetter(ctx), s.fieldGetter)
		if err != nil {
			return err
		}
		oldObj, err := dao.WithEqual(s.keyFieldName, key).First()
		if err != nil {
			return err
		}
		for _, onUpdateHdl := range s.onUpdate {
			onUpdateHdl(oldObj, obj)
		}
		if err := s.modify(ctx, dao, key, obj, func(dao *Dao[GormModelT, GenDoT], obj *GormModelT) (gen.ResultInfo, error) {
			var columns []string
			var updateRv bool
			if s.rvFieldName != "" {
				rv := util.GetStringField(obj, s.rvFieldOffset)
				if rv != "" {
					i, err := strconv.Atoi(rv)
					if err != nil {
						return gen.ResultInfo{}, fmt.Errorf("the revision must be number")
					}
					resourceVersion = strconv.Itoa(i + 1)
					util.SetStringField(obj, s.rvFieldOffset, resourceVersion)
					updateRv = true
				}
			}
			if updateRv {
				columns = s.columns
			} else {
				// don't update revision field
				for _, col := range s.columns {
					if col != s.rvFieldName {
						columns = append(columns, col)
					}
				}
			}
			result, err := dao.Select(columns).Updates(obj)
			return result, err
		}); err != nil {
			return err
		}

		return s.pubwatcher.Publish(ctx, watch.EventTypeUpdated, obj)
	})

	return
}

// Delete remove the object specified by key. If the key don't exists, it will
// return NotFound error
func (s *store[GormModelT, GenDoT]) Delete(ctx context.Context, key string, obj *GormModelT) (err error) {
	if obj == nil {
		obj = new(GormModelT)
	}
	util.SetStringField(obj, s.keyFieldOffset, key)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		dao, err := NewDao[GormModelT](s.genDaoGetter(ctx), s.fieldGetter)
		if err != nil {
			return err
		}
		if err := s.modify(ctx, dao, key, obj, func(dao *Dao[GormModelT, GenDoT], obj *GormModelT) (gen.ResultInfo, error) {
			result, err := dao.Returning(&obj, s.columns...).Delete(obj)
			return result, err
		}); err != nil {
			return err
		}
		return s.pubwatcher.Publish(ctx, watch.EventTypeDeleted, obj)
	})
	return
}

func (s *store[GormModelT, GenDoT]) Watch(ctx context.Context) (watch.Channel[GormModelT], error) {
	return s.pubwatcher.Watch(ctx)
}
