package types

// DimProperties represents a set of properties associated with a given
// dimension value
type Dimension struct {
	// Name of the dimension
	Name string
	// Value of the dimension
	Value string
	// Properties to be set on the dimension
	Properties map[string]string
	// Tags to apply to the dimension value
	Tags map[string]bool
	// Whether to merge the given properties/tags into an existing dimension,
	// or whether to entirely replace the set of properties/tags.
	// When this is true, you can delete tags by setting the value in Tags to
	// `false`, and you can delete properties by setting the value in
	// Properties to `nil`.
	MergeIntoExisting bool
}

// DimensionKey is what uniquely identifies a dimension, its name and value
// together.
type DimensionKey struct {
	Name  string
	Value string
}

// Key of the dimension, which is its name and value combined.
func (dp *Dimension) Key() DimensionKey {
	return DimensionKey{
		Name:  dp.Name,
		Value: dp.Value,
	}
}

// Copy creates a copy of the the given dimProps object
func (dp *Dimension) Copy() *Dimension {
	clonedProperties := make(map[string]string)
	for k, v := range dp.Properties {
		clonedProperties[k] = v
	}

	clonedTags := make(map[string]bool)
	for k, v := range dp.Tags {
		clonedTags[k] = v
	}

	return &Dimension{
		Name:       dp.Name,
		Value:      dp.Value,
		Properties: clonedProperties,
		Tags:       clonedTags,
	}
}
