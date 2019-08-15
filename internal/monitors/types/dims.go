package types

// Dimension represents a SignalFx dimension object and its associated
// properties and tags.
type Dimension struct {
	// Name of the dimension
	Name string
	// Value of the dimension
	Value string
	// Properties to be set on the dimension
	Properties map[string]string
	// Tags to apply to the dimension value
	Tags map[string]bool
	// Whether to do a query of existing dimension properties/tags and merge
	// the given values into those before updating, or whether to entirely
	// replace the set of properties/tags.
	MergeIntoExisting bool
}

// DimensionKey is what uniquely identifies a dimension, its name and value
// together.
type DimensionKey struct {
	Name  string
	Value string
}

func (d *Dimension) Key() DimensionKey {
	return DimensionKey{
		Name:  d.Name,
		Value: d.Value,
	}
}

// Copy creates a copy of the the given Dimension object
func (d *Dimension) Copy() *Dimension {
	clonedProperties := make(map[string]string)
	for k, v := range d.Properties {
		clonedProperties[k] = v
	}

	clonedTags := make(map[string]bool)
	for k, v := range d.Tags {
		clonedTags[k] = v
	}

	return &Dimension{
		Name:              d.Name,
		Value:             d.Value,
		Properties:        clonedProperties,
		Tags:              clonedTags,
		MergeIntoExisting: d.MergeIntoExisting,
	}
}
