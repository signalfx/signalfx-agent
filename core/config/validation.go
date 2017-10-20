package config

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
