package gorm

import (
	"fmt"
	"sync"

	"gorm.io/gen/field"
)

type FieldGetter interface {
	GetFieldByName(fieldName string) (field.OrderExpr, bool)
}

func NewFieldNotExistError(fieldName string) error {
	return fmt.Errorf("no such field '%s'", fieldName)
}

type SafeFieldGetter struct {
	mu          sync.RWMutex
	fieldGetter FieldGetter
}

func NewSafeFieldGetter(fieldGetter FieldGetter) *SafeFieldGetter {
	return &SafeFieldGetter{
		fieldGetter: fieldGetter,
	}
}

func (s *SafeFieldGetter) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fieldGetter.GetFieldByName(fieldName)
}
