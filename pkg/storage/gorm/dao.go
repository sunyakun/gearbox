package gorm

import (
	"fmt"

	"gorm.io/gen"
	"gorm.io/gen/field"
)

func NewNotImplementError(iface string) error {
	return fmt.Errorf("storage runtime error, the given type not implement %s", iface)
}

type GenDoInterface[T any] interface {
	Where(...gen.Condition) T
	Returning(value interface{}, columns ...string) T
	Select(conds ...field.Expr) T
	TableName() string
}

type GenDaoInterface[T any] interface {
	Create(values ...*T) error
	First() (*T, error)
	Find() ([]*T, error)
	FindByPage(offset int, limit int) (result []*T, count int64, err error)
	Updates(obj interface{}) (gen.ResultInfo, error)
	Delete(...*T) (info gen.ResultInfo, err error)
}

type returningInput struct {
	value   any
	columns []string
}

type Dao[GormModelT, GenDoT any] struct {
	genDo       GenDoInterface[GenDoT]
	genDao      GenDaoInterface[GormModelT]
	fieldGetter FieldGetter

	conditions     []gen.Condition
	returningInput *returningInput
	equalInput     [][]string
	selectInput    []string
}

func NewDao[GormModelT, GenDoT any](genDo GenDoT, fieldGetter FieldGetter) (*Dao[GormModelT, GenDoT], error) {
	var ok bool
	dao := &Dao[GormModelT, GenDoT]{fieldGetter: fieldGetter}
	dao.genDo, ok = interface{}(genDo).(GenDoInterface[GenDoT])
	if !ok {
		return nil, NewNotImplementError("GenDoInterface[T]")
	}
	return dao, nil
}

func (dao *Dao[GormModelT, GenDoT]) copy() *Dao[GormModelT, GenDoT] {
	return &Dao[GormModelT, GenDoT]{
		genDo:          dao.genDo,
		genDao:         dao.genDao,
		fieldGetter:    dao.fieldGetter,
		conditions:     dao.conditions,
		returningInput: dao.returningInput,
		equalInput:     dao.equalInput,
		selectInput:    dao.selectInput,
	}
}

func (dao *Dao[GormModelT, GenDoT]) WithEqual(key, val string) *Dao[GormModelT, GenDoT] {
	d := dao.copy()
	d.equalInput = append(dao.equalInput, []string{key, val})
	return d
}

func (dao *Dao[GormModelT, GenDoT]) Where(conds ...gen.Condition) *Dao[GormModelT, GenDoT] {
	d := dao.copy()
	d.conditions = append(dao.conditions, conds...)
	return d
}

func (dao *Dao[GormModelT, GenDoT]) Select(columns []string) *Dao[GormModelT, GenDoT] {
	d := dao.copy()
	d.selectInput = columns
	return d
}

func (dao *Dao[GormModelT, GenDoT]) Returning(value interface{}, columns ...string) *Dao[GormModelT, GenDoT] {
	d := dao.copy()
	d.returningInput = &returningInput{
		value:   value,
		columns: columns,
	}
	return d
}

func (dao *Dao[GormModelT, GenDoT]) prepareGenDao() error {
	var ok bool

	if len(dao.selectInput) != 0 {
		var selectedFields []field.Expr
		for _, column := range dao.selectInput {
			f, ok := dao.fieldGetter.GetFieldByName(column)
			if !ok {
				return NewFieldNotExistError(column)
			}
			selectedFields = append(selectedFields, f)

		}
		dao.genDo, ok = interface{}(dao.genDo.Select(selectedFields...)).(GenDoInterface[GenDoT])
		if !ok {
			return NewNotImplementError("GenDoInterface[T]")
		}
		dao.selectInput = []string{}
	}

	if len(dao.equalInput) != 0 {
		for _, item := range dao.equalInput {
			fieldName, val := item[0], item[1]
			f, ok := dao.fieldGetter.GetFieldByName(fieldName)
			if !ok {
				return NewFieldNotExistError(fieldName)
			}
			strfield, ok := f.(field.String)
			if !ok {
				return fmt.Errorf("field '%s' is not type string", fieldName)
			}
			dao.conditions = append(dao.conditions, strfield.Eq(val))
		}
		dao.equalInput = [][]string{}
	}

	if len(dao.conditions) != 0 {
		dao.genDo, ok = interface{}(dao.genDo.Where(dao.conditions...)).(GenDoInterface[GenDoT])
		if !ok {
			return NewNotImplementError("GenDoInterface[T]")
		}
		dao.conditions = []gen.Condition{}
	}

	if dao.returningInput != nil {
		dao.genDo, ok = interface{}(dao.genDo.Returning(dao.returningInput.value, dao.returningInput.columns...)).(GenDoInterface[GenDoT])
		if !ok {
			return NewNotImplementError("GenDoInterface[T]")
		}
		dao.returningInput = nil
	}

	dao.genDao, ok = interface{}(dao.genDo).(GenDaoInterface[GormModelT])
	if !ok {
		return NewNotImplementError("GenDaoInterface[T]")
	}

	return nil
}

func (dao *Dao[GormModelT, GenDoT]) Create(values ...*GormModelT) error {
	if err := dao.prepareGenDao(); err != nil {
		return err
	}
	return dao.genDao.Create(values...)
}

func (dao *Dao[GormModelT, GenDoT]) First() (*GormModelT, error) {
	if err := dao.prepareGenDao(); err != nil {
		return nil, err
	}
	return dao.genDao.First()
}

func (dao *Dao[GormModelT, GenDoT]) Find() ([]*GormModelT, error) {
	if err := dao.prepareGenDao(); err != nil {
		return nil, err
	}
	return dao.genDao.Find()
}

func (dao *Dao[GormModelT, GenDoT]) FindByPage(offset int, limit int) (result []*GormModelT, count int64, err error) {
	if err := dao.prepareGenDao(); err != nil {
		return nil, 0, err
	}
	return dao.genDao.FindByPage(offset, limit)
}

func (dao *Dao[GormModelT, GenDoT]) Updates(obj interface{}) (gen.ResultInfo, error) {
	if err := dao.prepareGenDao(); err != nil {
		return gen.ResultInfo{}, err
	}
	return dao.genDao.Updates(obj)
}

func (dao *Dao[GormModelT, GenDoT]) Delete(objs ...*GormModelT) (info gen.ResultInfo, err error) {
	if err := dao.prepareGenDao(); err != nil {
		return gen.ResultInfo{}, err
	}
	return dao.genDao.Delete(objs...)
}
