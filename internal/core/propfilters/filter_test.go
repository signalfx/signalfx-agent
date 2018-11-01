package propfilters

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/stretchr/testify/assert"
)

func TestFilters(t *testing.T) {
	t.Run("Filter a single property name", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Filter a regex property name", func(t *testing.T) {
		f, _ := New([]string{`/pod*/`}, []string{"*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Filter a globbed property name", func(t *testing.T) {
		f, _ := New([]string{`pod*`}, []string{"*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Filter a single property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"123"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Filter a regex property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{`/12*/`},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Filter a globbed property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"12*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Filter a property name and value", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"abc"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "abc", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(filteredProperties)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Filter a single dimension name", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"kubernetes_pod_uid"}, []string{"*"})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.True(t, f.MatchesDimension(dimensionName, dimensionValue))
		assert.False(t, f.MatchesDimension("kubernetes_node_name", dimensionValue))
	})

	t.Run("Filter a single dimension value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"*"}, []string{"789"})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.True(t, f.MatchesDimension(dimensionName, dimensionValue))
	})

	t.Run("Filter a regex dimension name", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{`/kubernetes_pod.*/`}, []string{"*"})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.True(t, f.MatchesDimension(dimensionName, dimensionValue))
		assert.False(t, f.MatchesDimension("kubernetes_node", dimensionValue))
	})

	t.Run("Filter a regex dimension value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"*"}, []string{`/7.*/`})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.True(t, f.MatchesDimension(dimensionName, dimensionValue))
		assert.False(t, f.MatchesDimension(dimensionName, "456"))
	})

	t.Run("Filter a globbed dimension name", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"kubernetes_pod*"}, []string{"*"})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.True(t, f.MatchesDimension(dimensionName, dimensionValue))
		assert.False(t, f.MatchesDimension("kubernetes_node", dimensionValue))
	})

	t.Run("Filter a globbed dimension value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"*"}, []string{"7*"})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.True(t, f.MatchesDimension(dimensionName, dimensionValue))
		assert.False(t, f.MatchesDimension(dimensionName, "456"))
	})

	t.Run("Filter a dimension name and value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"kubernetes_pod_uid"}, []string{"789"})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.True(t, f.MatchesDimension(dimensionName, dimensionValue))
		assert.False(t, f.MatchesDimension("kubernetes_node", dimensionValue))
		assert.False(t, f.MatchesDimension(dimensionName, "123"))
	})

	t.Run("Filter a dimprops object given dimension name", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"kubernetes_pod_uid"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "789",
			},
			Properties: properties,
			Tags:       nil,
		}

		spew.Dump(f)
		assert.Nil(t, f.FilterDimProps(dimProps))
	})

	t.Run("Filter a dimprops object given property name", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "789",
			},
			Properties: properties,
			Tags:       nil,
		}
		filteredDimProps := f.FilterDimProps(dimProps)

		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredDimProps.Properties, expectedProperties)
	})

	t.Run("Filter a dimprops object given property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"123"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "789",
			},
			Properties: properties,
			Tags:       nil,
		}

		filteredDimProps := f.FilterDimProps(dimProps)

		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc"}
		assert.Equal(t, filteredDimProps.Properties, expectedProperties)
	})

	t.Run("Filter a dimprops object given property name and value", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"123"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc", "service_uid": "123"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "789",
			},
			Properties: properties,
			Tags:       nil,
		}
		filteredDimProps := f.FilterDimProps(dimProps)
		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc", "service_uid": "123"}
		assert.Equal(t, filteredDimProps.Properties, expectedProperties)
	})

	t.Run("Filter a dimprops object given dimension value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"*"}, []string{"789"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "789",
			},
			Properties: properties,
			Tags:       nil,
		}

		spew.Dump(f)
		assert.Nil(t, f.FilterDimProps(dimProps))
	})

	t.Run("Filter a dimprops object given dimension name and property name", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"*"},
			[]string{"kubernetes_pod_uid"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc", "service_uid": "123"}
		nodeProperties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc", "service_uid": "123"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "789",
			},
			Properties: properties,
			Tags:       nil,
		}
		dimPropsNode := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_node",
				Value: "minikube",
			},
			Properties: nodeProperties,
			Tags:       nil,
		}
		filteredDimProps := f.FilterDimProps(dimProps)
		nodeFilteredDimProps := f.FilterDimProps(dimPropsNode)
		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc", "service_uid": "123"}
		assert.Equal(t, filteredDimProps.Properties, expectedProperties)
		assert.Equal(t, nodeFilteredDimProps.Properties, properties)
	})

	t.Run("Filter a dimprops object given dimension name, and property name and value", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"123"},
			[]string{"kubernetes_pod_uid"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc", "service_uid": "123"}
		nodeProperties := map[string]string{"pod-template-hash": "789", "replicaSet": "abc", "service_uid": "123"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "123",
			},
			Properties: properties,
			Tags:       nil,
		}
		dimPropsNode := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "789",
			},
			Properties: nodeProperties,
			Tags:       nil,
		}
		filteredDimProps := f.FilterDimProps(dimProps)
		nodeFilteredDimProps := f.FilterDimProps(dimPropsNode)
		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc", "service_uid": "123"}
		assert.Equal(t, filteredDimProps.Properties, expectedProperties)
		assert.Equal(t, nodeFilteredDimProps.Properties, nodeProperties)
	})

	t.Run("Filter a dimprops object given dimension name and value, and property name and value", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"123"},
			[]string{"kubernetes_pod_uid"}, []string{"123"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc", "service_uid": "123"}
		nodeProperties := map[string]string{"pod-template-hash": "789", "replicaSet": "abc", "service_uid": "123"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "123",
			},
			Properties: properties,
			Tags:       nil,
		}
		dimPropsNode := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "123",
			},
			Properties: nodeProperties,
			Tags:       nil,
		}
		filteredDimProps := f.FilterDimProps(dimProps)
		nodeFilteredDimProps := f.FilterDimProps(dimPropsNode)
		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc", "service_uid": "123"}
		assert.Equal(t, filteredDimProps.Properties, expectedProperties)
		assert.Equal(t, nodeFilteredDimProps.Properties, nodeProperties)
	})

	t.Run("Filter a dimprops object given dimension value, and property name and value", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"123"},
			[]string{"*"}, []string{"123"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc", "service_uid": "123"}
		nodeProperties := map[string]string{"pod-template-hash": "789", "replicaSet": "abc", "service_uid": "123"}
		dimProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "123",
			},
			Properties: properties,
			Tags:       nil,
		}
		dimPropsNode := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "kubernetes_pod_uid",
				Value: "123",
			},
			Properties: nodeProperties,
			Tags:       nil,
		}
		filteredDimProps := f.FilterDimProps(dimProps)
		nodeFilteredDimProps := f.FilterDimProps(dimPropsNode)
		spew.Dump(f)
		expectedProperties := map[string]string{"replicaSet": "abc", "service_uid": "123"}
		assert.Equal(t, filteredDimProps.Properties, expectedProperties)
		assert.Equal(t, nodeFilteredDimProps.Properties, nodeProperties)
	})

	// negation tests
	t.Run("Filter a single negated property name", func(t *testing.T) {
		f, _ := New([]string{"!pod-template-hash"}, []string{"*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		expectedProperties := map[string]string{"pod-template-hash": "123"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Filter a single negated property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"!123"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		expectedProperties := map[string]string{"pod-template-hash": "123"}
		assert.Equal(t, filteredProperties, expectedProperties)
	})

	t.Run("Match a negated dimension name", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"!kubernetes_pod_uid"}, []string{"*"})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.False(t, f.MatchesDimension(dimensionName, dimensionValue))
		assert.True(t, f.MatchesDimension("kubernetes_node_name", dimensionValue))
	})

	t.Run("Match a negated dimension value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"*"},
			[]string{"*"}, []string{"!789"})

		dimensionName := "kubernetes_pod_uid"
		dimensionValue := "789"

		spew.Dump(f)
		assert.False(t, f.MatchesDimension(dimensionName, dimensionValue))
		assert.True(t, f.MatchesDimension(dimensionName, "123"))
	})
}
