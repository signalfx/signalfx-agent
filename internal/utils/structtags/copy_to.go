package structtags

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	copyToTag = "copyTo"
)

// CopyTo -
func CopyTo(ptr interface{}) error {

	v := reflect.ValueOf(ptr).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		if val := t.Field(i).Tag.Get(copyToTag); val != "-" && val != "" {
			log.Info(val)
			var targets []string

			// initialize with tag value
			var groups = []string{val}

			// break apart targets and os commands
			if strings.Contains(val, ",GOOS=") {
				groups = strings.Split(val, ",GOOS=")
			}

			// break apart the targets
			targets = strings.Split(groups[0], ",")

			// check os eligibility
			OSEligible := true
			if len(groups) == 2 {
				OSEligible = isOSEligible(groups[1])
			}

			// if eligible copy the value to the targets
			if OSEligible {
				for _, target := range targets {
					sourceField := v.Field(i)
					targetField := v.FieldByName(target)
					if targetField.CanSet() && sourceField.Kind() == targetField.Kind() {
						targetField.Set(v.Field(i))
					} else {
						return fmt.Errorf("Unable to copy struct %v to target %s", sourceField, target)
					}
				}
			}
		}
	}
	return nil
}

// isOSEligible - determines if the os is eligible from the array of strings
func isOSEligible(OSString string) bool {
	// if the os string is empty
	if OSString == "" {
		return true
	}
	// check if the current os is explicitly excluded Ex. "!windows"
	if strings.Contains(OSString, fmt.Sprintf("!%s", runtime.GOOS)) {
		return false
	}
	// check if the os is explicitly included Ex. "windows"
	if strings.Contains(OSString, runtime.GOOS) {
		return true
	}
	// check for explicitly defined operating systems Ex. windows != "linux"
	operatingSystems := strings.Split(OSString, ",")
	for _, f := range operatingSystems {
		if !strings.Contains(f, "!") {
			return false
		}
	}
	// any explicitly listed oses were exclusionary
	// and the runtime operating system doesn't match any of them
	return true
}
