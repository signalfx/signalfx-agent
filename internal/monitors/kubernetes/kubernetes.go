package kubernetes

import (
	// Import the monitors so that they get registered
	_ "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/cluster"
	//_ "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/events"
)
