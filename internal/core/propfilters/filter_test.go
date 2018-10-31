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
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["replicaSet"] == "abc")
	})

	t.Run("Filter a regex property name", func(t *testing.T) {
		f, _ := New([]string{`/pod*/`}, []string{"*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["replicaSet"] == "abc")
	})

	t.Run("Filter a globbed property name", func(t *testing.T) {
		f, _ := New([]string{`pod*`}, []string{"*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["replicaSet"] == "abc")
	})

	t.Run("Filter a single property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"123"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["replicaSet"] == "abc")
	})

	t.Run("Filter a regex property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{`/12*/`},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["replicaSet"] == "abc")
	})

	t.Run("Filter a globbed property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"12*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["replicaSet"] == "abc")
	})

	t.Run("Filter a property name and value", func(t *testing.T) {
		f, _ := New([]string{"pod-template-hash"}, []string{"abc"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "abc", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(filteredProperties)
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["replicaSet"] == "abc")
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
		assert.True(t, f.FilterDimProps(dimProps) == nil)
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
		assert.True(t, len(filteredDimProps.Properties) == 1)
		assert.True(t, filteredDimProps.Properties["replicaSet"] == "abc")
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
		assert.True(t, len(filteredDimProps.Properties) == 1)
		assert.True(t, filteredDimProps.Properties["replicaSet"] == "abc")
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
		assert.True(t, f.FilterDimProps(dimProps) == nil)
	})

	// negation tests
	t.Run("Filter a single negated property name", func(t *testing.T) {
		f, _ := New([]string{"!pod-template-hash"}, []string{"*"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["pod-template-hash"] == "123")
	})

	t.Run("Filter a single negated property value", func(t *testing.T) {
		f, _ := New([]string{"*"}, []string{"!123"},
			[]string{"*"}, []string{"*"})

		properties := map[string]string{"pod-template-hash": "123", "replicaSet": "abc"}
		filteredProperties := f.FilterProperties(properties)

		spew.Dump(f)
		assert.True(t, len(filteredProperties) == 1)
		assert.True(t, filteredProperties["pod-template-hash"] == "123")
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

}
