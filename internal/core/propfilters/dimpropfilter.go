// Package propfilters has logic describing the filtering of unwanted properties.  Filters
// are configured from the agent configuration file and is intended to be passed
// into each monitor for use if it sends propeties on its own.
package propfilters

import (
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
)

// DimPropsFilter is designed to filter SignalFx property objects.  It
// can filter based on property names, property values, dimension names,
// and dimension values. It also supports both static, globbed, and regex
// patterns for filter values.
// All properties matched will be filtered.
type DimPropsFilter interface {
	// Filters out properties from DimProperties object
	FilterDimProps(dimProps *types.DimProperties) *types.DimProperties
	MatchesDimension(name string, value string) bool
	FilterProperties(properties map[string]string) map[string]string
}

// basicDimPropsFilter is an implementation of DimPropsFilter
type basicDimPropsFilter struct {
	propertyNameFilter   filter.StringFilter
	propertyValueFilter  filter.StringFilter
	dimensionNameFilter  filter.StringFilter
	dimensionValueFilter filter.StringFilter
}

// New returns a new filter with the given configuration
func New(propertyNames []string, propertyValues []string, dimensionNames []string,
	dimensionValues []string) (DimPropsFilter, error) {

	var propertyNameFilter filter.StringFilter
	if len(propertyNames) > 0 {
		var err error
		propertyNameFilter, err = filter.NewStringFilter(propertyNames)
		if err != nil {
			return nil, err
		}
	}

	var propertyValueFilter filter.StringFilter
	if len(propertyValues) > 0 {
		var err error
		propertyValueFilter, err = filter.NewStringFilter(propertyValues)
		if err != nil {
			return nil, err
		}
	}

	var dimensionNameFilter filter.StringFilter
	if len(dimensionNames) > 0 {
		var err error
		dimensionNameFilter, err = filter.NewStringFilter(dimensionNames)
		if err != nil {
			return nil, err
		}
	}

	var dimensionValueFilter filter.StringFilter
	if len(dimensionValues) > 0 {
		var err error
		dimensionValueFilter, err = filter.NewStringFilter(dimensionValues)
		if err != nil {
			return nil, err
		}
	}

	return &basicDimPropsFilter{
		propertyNameFilter:   propertyNameFilter,
		propertyValueFilter:  propertyValueFilter,
		dimensionNameFilter:  dimensionNameFilter,
		dimensionValueFilter: dimensionValueFilter,
	}, nil
}

// Filter applies the filter to the given DimProperties and returns a new
// filtered DimProperties
func (f *basicDimPropsFilter) FilterDimProps(dimProps *types.DimProperties) *types.DimProperties {
	if dimProps == nil {
		return nil
	}
	filteredProperties := make(map[string]string, len(dimProps.Properties))

	if f.MatchesDimension(dimProps.Name, dimProps.Value) {
		filteredProperties = f.FilterProperties(dimProps.Properties)
	} else {
		filteredProperties = dimProps.Properties
	}

	if len(filteredProperties) > 0 {
		return &types.DimProperties{
			Dimension:  dimProps.Dimension,
			Properties: filteredProperties,
			Tags:       dimProps.Tags,
		}
	}

	return nil
}

// FilterProperties uses the propertyNameFilter and propertyValueFilter given to
// filter out properties in a map if either the name or value matches
func (f *basicDimPropsFilter) FilterProperties(properties map[string]string) map[string]string {
	filteredProperties := make(map[string]string, len(properties))
	for propName, propValue := range properties {
		if (!f.propertyNameFilter.Matches(propName)) ||
			(!f.propertyValueFilter.Matches(propValue)) {
			filteredProperties[propName] = propValue
		}
	}

	return filteredProperties
}

// MatchesDimension checks both dimensionNameFilter and dimensionValueFilter
// and if both match, returns true
func (f *basicDimPropsFilter) MatchesDimension(name string, value string) bool {
	return f.dimensionNameFilter.Matches(name) && f.dimensionValueFilter.Matches(value)
}
