package correlations

import "net/http"

// Type is the type of correlation
type Type string

const (
	// Service is for correlating services
	Service Type = "service"
	// Environment is for correlating environments
	Environment Type = "environment"
)

// Operation is the type of operation associated with a correlation
type Operation string

const (
	// Put is for HTTP PUT operations
	Put Operation = http.MethodPut
	// Delete is for HTTP DELETE operations
	Delete Operation = http.MethodDelete
)

// Correlation is a struct referencing
type Correlation struct {
	// Type is the type of correlation
	Type Type
	// Operation is the operation associated with the correlation
	Operation Operation
	// DimName is the dimension name
	DimName string
	// DimValue is the dimension value
	DimValue string
	// Value is the value to correlate with the DimName and DimValue
	Value string
}
