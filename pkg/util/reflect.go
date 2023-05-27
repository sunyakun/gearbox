package util

import (
	"fmt"
	"reflect"
	"unsafe"

	"gorm.io/gorm/schema"
)

func checkStruct(rt reflect.Type) error {
	if rt.Kind() != reflect.Struct {
		return fmt.Errorf("'%s' is not a struct", rt.Name())
	}
	return nil
}

func checkNondefined(rt reflect.Type) error {
	if rt.Name() == "" {
		return fmt.Errorf("non-defined type is not allow")
	}
	return nil
}

func ReflectDefinedStruct[T any]() (reflect.Type, error) {
	var t T
	rt := reflect.TypeOf(t)

	if err := checkNondefined(rt); err != nil {
		return nil, err
	}

	if err := checkStruct(rt); err != nil {
		return nil, err
	}

	return rt, nil
}

// GetStringField will use unsafe to get the struct field, please use this carefully,
// make sure the type T is a struct and the offset will not exceed in the struct's boundary.
func GetStringField[T any](structptr *T, offset uintptr) string {
	f := unsafe.Add(unsafe.Pointer(structptr), offset)
	s := *(*string)(f)
	return s
}

// SetStringField will use unsafe to set the struct field, please use this carefully,
// make sure the type T is a struct and the offset will not exceed in the struct's boundary.
func SetStringField[T any](structptr *T, offset uintptr, val string) {
	f := unsafe.Add(unsafe.Pointer(structptr), offset)
	sptr := (*string)(f)
	*sptr = val
}

// InspectColumns collect all fields with tag "column:<columnName>" and without tag "autoIncrement:true"
func InspectColumns(rt reflect.Type) []string {
	var columns []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		tagSetting := schema.ParseTagSetting(f.Tag.Get("gorm"), ";")
		if tagSetting["COLUMN"] != "" && tagSetting["AUTOINCREMENT"] != "true" {
			columns = append(columns, tagSetting["COLUMN"])
		}
	}
	return columns
}

// GetFieldByGormColumnTag searched the struct fields and find the field with tag "column:<columnName>"
func GetFieldByGormColumnTag(rt reflect.Type, columnName string) (reflect.StructField, bool) {
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		tagSetting := schema.ParseTagSetting(f.Tag.Get("gorm"), ";")
		if tagSetting["COLUMN"] == columnName {
			return f, true
		}
	}
	return reflect.StructField{}, false
}
