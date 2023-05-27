package selector

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Operator = selection.Operator

type Requirement = labels.Requirement

type PathOption = field.PathOption

func Parse(selector string, opts ...PathOption) ([]Requirement, error) {
	return labels.ParseToRequirements(selector, opts...)
}

func NewRequirement(key string, op Operator, vals []string, opts ...PathOption) (*Requirement, error) {
	return labels.NewRequirement(key, op, vals, opts...)
}

const (
	DoesNotExist Operator = selection.DoesNotExist
	Equals       Operator = selection.Equals
	DoubleEquals Operator = selection.DoubleEquals
	In           Operator = selection.In
	NotEquals    Operator = selection.NotEquals
	NotIn        Operator = selection.NotIn
	Exists       Operator = selection.Exists
	GreaterThan  Operator = selection.GreaterThan
	LessThan     Operator = selection.LessThan
)
