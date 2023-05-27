package apis

import (
	"fmt"
	"reflect"
	"sync"
)

type Scheme struct {
	mu         sync.Mutex
	typeToKind map[reflect.Type]string
	KindToType map[string]reflect.Type
}

func NewScheme() *Scheme {
	return &Scheme{
		typeToKind: map[reflect.Type]string{},
		KindToType: map[string]reflect.Type{},
	}
}

func (s *Scheme) AddKnownTypes(types ...Object) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range types {
		rt := reflect.TypeOf(t)
		if rt.Kind() != reflect.Pointer {
			return fmt.Errorf("all types must be pointer")
		}
		rt = rt.Elem()
		if _, ok := s.KindToType[rt.Name()]; ok {
			return fmt.Errorf("type %q already added", rt.Name())
		}
		s.KindToType[rt.Name()] = rt
		s.typeToKind[rt] = rt.Name()
	}
	return nil
}

func (s *Scheme) AllKnownTypes() map[string]reflect.Type {
	var allKnownTypes = map[string]reflect.Type{}
	for name, t := range s.KindToType {
		allKnownTypes[name] = t
	}
	return allKnownTypes
}

func (s *Scheme) ObjectKind(obj Object) (string, error) {
	rt := reflect.TypeOf(obj)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	kind, ok := s.typeToKind[rt]
	if !ok {
		return "", fmt.Errorf("%q not registered", rt.Name())
	}
	return kind, nil
}
