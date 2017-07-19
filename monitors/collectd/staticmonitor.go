package collectd

import (
	"text/template"
)

// This is intended to be embedded in the individual monitors that represent
// non-dynamically configured (static) collectd plugins (e.g. the metadata
// plugin)
type StaticMonitorCore struct {
	BaseMonitor
}

func NewStaticMonitorCore(template *template.Template) *StaticMonitorCore {
	return &StaticMonitorCore{
		BaseMonitor: *NewBaseMonitor(template),
	}
}
