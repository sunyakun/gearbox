package gorm

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sunyakun/gearbox/pkg/storage/selector"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

type Field[T any] interface {
	Eq(value T) field.Expr
	Neq(T) field.Expr
	In(...T) field.Expr
	NotIn(...T) field.Expr
	Gt(T) field.Expr
	Lt(T) field.Expr
}

type BoolField interface {
	Is(bool) field.Expr
}

func atoi[T int | int8 | int16 | int32 | int64](s string) (T, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return T(v), nil
}

type Selector struct {
	fieldGetter FieldGetter
	parseToTime func(string) (time.Time, error)
}

func NewSelector(fieldGetter FieldGetter, parseToTime func(string) (time.Time, error)) *Selector {
	return &Selector{
		fieldGetter: fieldGetter,
		parseToTime: parseToTime,
	}
}

func (s *Selector) generateExpr(requirement selector.Requirement) (field.Expr, error) {
	fieldName := requirement.Key()

	targetField, ok := s.fieldGetter.GetFieldByName(fieldName)
	if !ok {
		return nil, NewFieldNotExistError(fieldName)
	}

	switch f := targetField.(type) {
	case field.String:
		return generateExprByField(requirement, f, func(input string) (string, error) { return input, nil })
	case field.Int:
		return generateExprByField(requirement, f, atoi[int])
	case field.Int8:
		return generateExprByField(requirement, f, atoi[int8])
	case field.Int16:
		return generateExprByField(requirement, f, atoi[int16])
	case field.Int32:
		return generateExprByField(requirement, f, atoi[int32])
	case field.Int64:
		return generateExprByField(requirement, f, atoi[int64])
	case field.Bool:
		return generateExprByField(requirement, f, strconv.ParseBool)
	case field.Float32:
		return generateExprByField(requirement, f, func(s string) (float32, error) {
			v, err := strconv.ParseFloat(s, 32)
			return float32(v), err
		})
	case field.Float64:
		return generateExprByField(requirement, f, func(s string) (float64, error) {
			v, err := strconv.ParseFloat(s, 64)
			return v, err
		})
	case field.Time:
		return generateExprByField(requirement, f, s.parseToTime)
	default:
		return nil, fmt.Errorf("don't known how to apply the selector '%s' to the underlying storage", requirement.String())
	}
}

func (s *Selector) GenerateConditions(requirements []selector.Requirement) ([]gen.Condition, error) {
	conditions := make([]gen.Condition, 0)
	for _, requirement := range requirements {
		expr, err := s.generateExpr(requirement)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, expr)
	}
	return conditions, nil
}

func generateExprByField[T any](
	requirement selector.Requirement,
	f any,
	fromString func(string) (T, error),
) (field.Expr, error) {
	var fnmap = map[selector.Operator]any{}

	genericField, ok := f.(Field[T])
	if ok {
		fnmap = map[selector.Operator]any{
			selector.Equals:       genericField.Eq,
			selector.DoubleEquals: genericField.Eq,
			selector.In:           genericField.In,
			selector.NotEquals:    genericField.Neq,
			selector.NotIn:        genericField.NotIn,
			selector.GreaterThan:  genericField.Gt,
			selector.LessThan:     genericField.Lt,
		}
	}

	boolField, ok := f.(BoolField)
	if ok {
		fnmap = map[selector.Operator]any{
			selector.Equals: boolField.Is,
		}
	}

	opFn, ok := fnmap[requirement.Operator()]
	if !ok {
		return nil, fmt.Errorf("the underlying storage don't support operator '%s' for '%s'", requirement.Operator(), requirement.Key())
	}
	switch requirement.Operator() {
	case selector.Equals, selector.DoubleEquals, selector.NotEquals, selector.LessThan, selector.GreaterThan:
		fn, ok := opFn.(func(T) field.Expr)
		if !ok {
			return nil, fmt.Errorf("invalid function for '%s'", requirement.Operator())
		}
		value, ok := requirement.Values().PopAny()
		if !ok {
			return nil, fmt.Errorf("the value can't be empty for operator '%s'", requirement.Operator())
		}
		v, err := fromString(value)
		if err != nil {
			return nil, err
		}
		return fn(v), nil
	case selector.In, selector.NotIn:
		fn, ok := opFn.(func(...T) field.Expr)
		if !ok {
			return nil, fmt.Errorf("invalid function for '%s'", requirement.Operator())
		}
		values := requirement.Values().List()

		var err error
		result := make([]T, len(values))
		for idx := range values {
			result[idx], err = fromString(values[idx])
			if err != nil {
				return nil, err
			}
		}

		return fn(result...), nil
	case selector.Exists, selector.DoesNotExist:
		fn, ok := opFn.(func() field.Expr)
		if !ok {
			return nil, fmt.Errorf("")
		}
		return fn(), nil
	default:
		return nil, fmt.Errorf("unknown operator '%s'", requirement.Operator())
	}
}
