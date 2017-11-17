package utils

import "reflect"

// CloneInterface takes an object and returns a copy of it regardless of
// whether it is really a pointer underneath or not.  It is roughly equivalent
// to the following:
// b = *a  (if 'a' is a pointer)
// b = a (if 'a' is not a pointer)
func CloneInterface(a interface{}) interface{} {
	va := reflect.ValueOf(a)
	indirect := reflect.Indirect(va)
	new := reflect.New(indirect.Type())
	new.Elem().Set(reflect.ValueOf(indirect.Interface()))
	if va.Kind() == reflect.Ptr {
		return new.Interface()
	}
	return new.Elem().Interface()
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

// FindFieldWithEmbeddedStructs will look for a field with the given name,
// recursing down into embedded structs if there are any.
func FindFieldWithEmbeddedStructs(st interface{}, name string, type_ reflect.Type) reflect.Value {
	instanceValue := reflect.Indirect(reflect.ValueOf(st))
	fieldValue := instanceValue.FieldByName(name)

	if !fieldValue.IsValid() || fieldValue.Type() != type_ {
		embeddedValues := make([]reflect.Value, 0)

		for i := 0; i < instanceValue.Type().NumField(); i++ {
			field := instanceValue.Type().Field(i)
			if field.Type.Kind() == reflect.Struct && field.Anonymous && instanceValue.Field(i).CanSet() {
				embeddedValues = append(embeddedValues,
					FindFieldWithEmbeddedStructs(instanceValue.Field(i).Interface(), name, type_))
			}
		}
		for _, v := range embeddedValues {
			if v.IsValid() {
				return v
			}
		}
		return reflect.ValueOf(nil)
	}
	return fieldValue
}
