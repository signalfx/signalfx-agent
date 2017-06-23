package userconfig

// Label - stores labels and label values
type Label struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value,omitempty"`
}
