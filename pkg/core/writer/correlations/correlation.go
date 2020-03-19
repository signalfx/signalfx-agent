package correlations

// Type is the type of correlation
type Type string

const (
	// Service is for correlating services
	Service Type = "service"
	// Environment is for correlating environments
	Environment Type = "environment"
)

// Correlation is a struct referencing
type Correlation struct {
	// Type is the type of correlation
	Type Type
	// DimName is the dimension name
	DimName string
	// DimValue is the dimension value
	DimValue string
	// Value is the value to makeRequest with the DimName and DimValue
	Value string
}
