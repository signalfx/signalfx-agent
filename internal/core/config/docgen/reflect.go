package docgen

import (
	"fmt"
	"reflect"
	"strings"
)

func getYAMLName(f reflect.StructField) string {
	yamlTag := f.Tag.Get("yaml")
	return strings.SplitN(yamlTag, ",", 2)[0]
}

func getDefault(f reflect.StructField) string {
	defTag := f.Tag.Get("default")
	if defTag != "" {
		return defTag
	}
	if f.Type.Kind() == reflect.Ptr {
		return ""
	}
	return fmt.Sprintf("%v", reflect.Zero(f.Type).Interface())
}

func getRequired(f reflect.StructField) bool {
	validate := f.Tag.Get("validate")
	for _, v := range strings.Split(validate, ",") {
		if v == "required" {
			return true
		}
	}
	return false
}

// The kind with any pointer removed
func indirectKind(t reflect.Type) reflect.Kind {
	kind := t.Kind()
	if kind == reflect.Ptr {
		return t.Elem().Kind()
	}
	return kind
}

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}
