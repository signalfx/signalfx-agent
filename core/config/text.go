package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/signalfx/neo-agent/utils"
	yaml "gopkg.in/yaml.v2"
)

// ToString converts a config struct to a pseudo-yaml text outut.  If a struct
// field has the 'neverLog' tag, its value will be replaced by asterisks, or
// completely omitted if the tag value is 'omit'.
func ToString(conf interface{}) string {
	if conf == nil {
		return ""
	}

	var out string
	confValue := reflect.Indirect(reflect.ValueOf(conf))
	confStruct := confValue.Type()

	for i := 0; i < confStruct.NumField(); i++ {
		field := confStruct.Field(i)

		// PkgPath is empty only for exported fields, so it it's non-empty the
		// field is private
		if field.PkgPath != "" {
			continue
		}

		fieldName := utils.YAMLNameOfField(field)

		if fieldName == "" {
			continue
		}

		isStruct := field.Type.Kind() == reflect.Struct
		isPtrToStruct := field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct
		if isStruct || isPtrToStruct {
			// Flatten embedded struct's representation
			if field.Anonymous {
				out += ToString(confValue.Field(i).Interface())
				continue
			}

			out += fieldName + ":\n"
			out += utils.IndentLines(ToString(confValue.Field(i).Interface()), 2)
			continue
		}

		neverLogVal, neverLogPresent := field.Tag.Lookup("neverLog")
		var val string
		if neverLogPresent {
			if neverLogVal == "omit" {
				continue
			}
			field.Type = reflect.PtrTo(reflect.ValueOf("").Type())
			if reflect.Zero(field.Type).Interface() == confValue.Field(i).Interface() {
				val = ""
			} else {
				val = "***************"
			}
		} else {
			asYaml, _ := yaml.Marshal(confValue.Field(i).Interface())
			val = strings.Trim(string(asYaml), "\n")
		}

		separator := " "
		if strings.Contains(val, "\n") {
			separator = "\n"
			val = utils.IndentLines(val, 2)
		}

		out += fmt.Sprintf("%s:%s%s\n", fieldName, separator, val)
	}
	return out
}
