package splunk

import "github.com/signalfx/golib/v3/event"

// Just a dummy interface that covers all types of HEC inputs
type logEntry interface{}

type eventdata struct {
	Category   event.Category    `json:"category"`
	EventType  string            `json:"eventType"`
	Meta       map[string]string `json:"meta"`
	Dimensions map[string]string `json:"dimensions"`
	Properties map[string]string `json:"properties"`
}

// This is the format that the HEC input accepts
type logMetric struct {
	Time       int64             `json:"time"`                 // epoch time
	Host       string            `json:"host"`                 // hostname
	Source     string            `json:"source,omitempty"`     // optional description of the source of the event; typically the app's name
	SourceType string            `json:"sourcetype,omitempty"` // optional name of a Splunk parsing configuration; this is usually inferred by Splunk
	Index      string            `json:"index,omitempty"`      // optional name of the Splunk index to store the event in; not required if the token has a default index set in Splunk
	Event      string            `json:"event"`                // type of event: this is a metric.
	Fields     map[string]string `json:"fields"`               // metric data
}

// This is the format that the HEC input accepts
type logEvent struct {
	Time       int64     `json:"time"`                 // epoch time
	Host       string    `json:"host"`                 // hostname
	Source     string    `json:"source,omitempty"`     // optional description of the source of the event; typically the app's name
	SourceType string    `json:"sourcetype,omitempty"` // optional name of a Splunk parsing configuration; this is usually inferred by Splunk
	Index      string    `json:"index,omitempty"`      // optional name of the Splunk index to store the event in; not required if the token has a default index set in Splunk
	Event      eventdata `json:"event"`                // event data
}
