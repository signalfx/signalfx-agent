package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/signalfx/signalfx-agent/internal/utils"
	validator "gopkg.in/go-playground/validator.v9"
)

// Validatable should be implemented by config structs that want to provide
// validation when the config is loaded.
type Validatable interface {
	Validate() error
}

// ValidateCustomConfig for module-specific config ahead of time for a specific
// module configuration.  This way, the Configure method of modules will be
// guaranteed to receive valid configuration.  The module-specific
// configuration struct must implement the Validate method that returns a bool.
func ValidateCustomConfig(conf interface{}) error {
	if v, ok := conf.(Validatable); ok {
		return v.Validate()
	}
	return nil
}

// ValidateStruct uses the `validate` struct tags to do standard validation
func ValidateStruct(confStruct interface{}) error {
	validate := validator.New()
	err := validate.Struct(confStruct)
	if err != nil {
		if ves, ok := err.(validator.ValidationErrors); ok {
			var msgs []string
			for _, e := range ves {
				fieldName := utils.YAMLNameOfFieldInStruct(e.Field(), confStruct)
				msgs = append(msgs, fmt.Sprintf("Validation error in field '%s': %s", fieldName, e.Tag()))
			}
			return errors.New(strings.Join(msgs, "; "))
		}
		return err
	}
	return nil
}
