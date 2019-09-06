package metrics

import (
	"fmt"
	"regexp"

	"github.com/iancoleman/strcase"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	v1 "k8s.io/api/core/v1"
)

func datapointsForNode(
	node *v1.Node,
	nodeConditionTypesToReport []string,
) []*datapoint.Datapoint {
	dims := map[string]string{
		"kubernetes_node":     node.Name,
		"kubernetes_node_uid": string(node.UID),
	}

	datapoints := make([]*datapoint.Datapoint, 0)
	for _, nodeConditionTypeValue := range nodeConditionTypesToReport {
		nodeConditionMetric := fmt.Sprintf("kubernetes.node_%s", strcase.ToSnake(nodeConditionTypeValue))
		v1NodeConditionTypeValue := v1.NodeConditionType(nodeConditionTypeValue)
		datapoints = append(
			datapoints,
			sfxclient.Gauge(
				nodeConditionMetric, dims, nodeConditionValue(node, v1NodeConditionTypeValue),
			),
		)
	}
	return datapoints
}

func dimPropsForNode(node *v1.Node) *atypes.DimProperties {
	props, tags := k8sutil.PropsAndTagsFromLabels(node.Labels)

	props["kubernetes_node"] = node.Name

	taintProps := getPropsFromTaints(node.Spec.Taints)
	for k, v := range taintProps {
		props[k] = v
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "kubernetes_node_uid",
			Value: string(node.UID),
		},
		Properties: props,
		Tags:       tags,
	}
}

func getPropsFromTaints(taints []v1.Taint) map[string]string {
	unsupportedPattern := regexp.MustCompile("[^a-zA-Z0-9_-]")

	props := make(map[string]string)

	for _, t := range taints {
		keyValueCombo := "taint"
		if len(t.Key) > 0 {
			keyValueCombo += ("_" + t.Key)
		}
		if len(t.Value) > 0 {
			keyValueCombo += ("_" + t.Value)
		}
		keyValueCombo = unsupportedPattern.ReplaceAllString(keyValueCombo, "_")

		if _, exists := props[keyValueCombo]; exists {
			props[keyValueCombo] += ("," + string(t.Effect))
		} else {
			props[keyValueCombo] = string(t.Effect)
		}
	}

	return props
}

var nodeConditionValues = map[v1.ConditionStatus]int64{
	v1.ConditionTrue:    1,
	v1.ConditionFalse:   0,
	v1.ConditionUnknown: -1,
}

func nodeConditionValue(node *v1.Node, condType v1.NodeConditionType) int64 {
	status := v1.ConditionUnknown
	for _, c := range node.Status.Conditions {
		if c.Type == condType {
			status = c.Status
			break
		}
	}
	return nodeConditionValues[status]
}
