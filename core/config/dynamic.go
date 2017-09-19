package config

import (
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/creasty/defaults"
	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

// DecodeOtherConfig will pull out the OtherConfig values from both
// ObserverConfig and MonitorConfig and decode them to a struct that is
// provided in the `out` arg.  If any values are not decoded, it is considered
// an error since the user provided config that will not be used and probably
// thought would.  Any errors decoding will cause `out` to be nil.
func DecodeOtherConfig(in CustomConfigurable, out interface{}) error {
	pkgPaths := strings.Split(reflect.Indirect(reflect.ValueOf(out)).Type().PkgPath(), "/")

	otherYaml, err := yaml.Marshal(in.GetOtherConfig())
	if err != nil {
		return err
	}

	err = yaml.UnmarshalStrict(otherYaml, out)
	if err != nil {
		log.WithFields(log.Fields{
			"package":     pkgPaths[len(pkgPaths)-1],
			"otherConfig": spew.Sdump(in.GetOtherConfig()),
			"error":       err,
		}).Error("Invalid module-specific configuration")
		return err
	}

	if err := defaults.Set(out); err != nil {
		log.WithFields(log.Fields{
			"package": pkgPaths[len(pkgPaths)-1],
			"error":   err,
			"out":     spew.Sdump(out),
		}).Error("Could not set defaults on module-specific config")
		return err
	}
	return nil
}

// FillInConfigTemplate takes a config template value that a monitor/observer
// provided and fills it in dynamically from the provided conf
func FillInConfigTemplate(embeddedFieldName string, configTemplate interface{}, conf CustomConfigurable) bool {
	templateValue := reflect.ValueOf(configTemplate)
	pkg := templateValue.Type().PkgPath()

	if templateValue.Kind() != reflect.Ptr || templateValue.Elem().Kind() != reflect.Struct {
		log.WithFields(log.Fields{
			"package": pkg,
			"kind":    templateValue.Kind(),
			"type":    templateValue.Type(),
		}).Error("Config template must be a pointer to a struct")
		return false
	}

	embeddedField := templateValue.Elem().FieldByName(embeddedFieldName)
	if !embeddedField.IsValid() {
		log.WithFields(log.Fields{
			"fieldName": embeddedFieldName,
			"fields":    utils.GetStructFieldNames(templateValue),
		}).Error("Could not find embedded field in config")
		return false
	}
	embeddedField.Set(reflect.Indirect(reflect.ValueOf(conf)))

	if err := DecodeOtherConfig(conf, configTemplate); err != nil {
		return false
	}
	if configTemplate == nil {
		return false
	}
	return true
}

// CallConfigure will call the Configure method on an observer or monitor with
// a `conf` object, typed to the correct type.  This allows monitors/observers
// to set the type of the config object to their own config and not have to
// worry about casting or converting.
func CallConfigure(instance, conf interface{}) bool {
	instanceVal := reflect.ValueOf(instance)
	_type := instanceVal.Type().PkgPath()

	confVal := reflect.ValueOf(conf)

	method := instanceVal.MethodByName("Configure")
	if !method.IsValid() {
		log.WithFields(log.Fields{
			"observerType": _type,
		}).Error("No Configure method found!")
		return false
	}

	if method.Type().NumIn() != 1 {
		log.WithFields(log.Fields{
			"observerType": _type,
			"numIn":        method.Type().NumIn(),
			"methodInType": method.Type().In(0),
			"confValType":  confVal.Type(),
		}).Error("Configure method should take exactly one argument that matches " +
			"the type of the config template provided in the Register function!")
		return false
	}

	if method.Type().NumOut() != 1 || method.Type().Out(0).Kind() != reflect.Bool {
		log.WithFields(log.Fields{
			"observerType": _type,
		}).Error("Configure method should return a bool")
		return false
	}

	return method.Call([]reflect.Value{confVal})[0].Bool()
}
