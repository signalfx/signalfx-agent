package selfdescribe

import (
	"encoding/json"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Only works if there is an explicit "yaml" struct tag
func getYAMLName(f reflect.StructField) string {
	yamlTag := f.Tag.Get("yaml")
	return strings.SplitN(yamlTag, ",", 2)[0]
}

// Assumes monitors are using the defaults package
func getDefault(f reflect.StructField) interface{} {
	if getRequired(f) {
		return nil
	}
	defTag := f.Tag.Get("default")
	if defTag != "" {
		var out interface{}
		err := json.Unmarshal([]byte(defTag), &out)
		if err != nil && (strings.HasPrefix(defTag, "{") || strings.HasPrefix(defTag, "[")) {
			log.WithError(err).Errorf("Could not unmarshal default value `%s` for field %s", defTag, f.Name)
			return defTag
		}
		return out
	}
	if f.Type.Kind() == reflect.Ptr {
		return nil
	}
	return reflect.Zero(f.Type).Interface()
}

// Assumes that monitors are using the validate package to do validation
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

// The type with any pointers removed
func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}
