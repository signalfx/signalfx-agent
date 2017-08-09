package utils

import "reflect"

// CloneInterface takes a clone and returns a copy of it regardless of whether
// it is really a pointer underneath or not.
func CloneInterface(a interface{}) interface{} {
	va := reflect.ValueOf(a)
	indirect := reflect.Indirect(va)
	new := reflect.New(indirect.Type())
	new.Elem().Set(reflect.ValueOf(indirect.Interface()))
	if va.Kind() == reflect.Ptr {
		return new.Interface()
	} else {
		return new.Interface()
	}
}

// GetStructFieldNames returns a slice with the names of all of the fields in
// the struct `s`.  This will panic if `s` is not a struct.
func GetStructFieldNames(s interface{}) []string {
	v := reflect.Indirect(reflect.ValueOf(s))
	out := []string{}

	for i := 0; i < v.Type().NumField(); i++ {
		out = append(out, v.Type().Field(i).Name)
	}

	return out
}
