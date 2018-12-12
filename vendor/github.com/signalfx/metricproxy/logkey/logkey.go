package logkey

import "github.com/signalfx/golib/log"

// ignored exists so that I can get some kind of coverage for this package
func ignored() string {
	// ignored
	return ""
}

var (
	// Filename is a system file name
	Filename = log.Key("filename")
	// Delta is the diff between two things, generally time
	Delta = log.Key("delta")
	// Dir is a directory
	Dir = log.Key("directory")
	// Struct is the name of a go struct
	Struct = log.Key("struct")
	// SHA1 is a SHA1 hash of something
	SHA1 = log.Key("sha1")
	// Config is the config passed to metricproxy
	Config = log.Key("config")
	// StatKeepers are a list of datapoint keepers
	StatKeepers = log.Key("stat_keepers")
	// RemoteAddr of a network connection
	RemoteAddr = log.Key("remote_addr")
	// CarbonLine is a direct line received from carbon protocol
	CarbonLine = log.Key("carbon_line")
	// Capacity is a size
	Capacity = log.Key("capacity")
	// TotalPipeline is the total number of things buffered and downstream of a forwarder
	TotalPipeline = log.Key("total_pipeline")
	// Protocol is the type of connection (signalfx/collectd/etc)
	Protocol = log.Key("protocol")
	// DebugAddr is the local address of a debug server
	DebugAddr = log.Key("debug_addr")
	// Env is environment variables
	Env = log.Key("env")
	// Time is the localtime of the log statement
	Time = log.Key("time")
	// Caller is the filename/line number of the calling function
	Caller = log.Key("caller")
	// Direction is either listening or sending
	Direction = log.Key("direction")
	// ForwardTo is where the data is going
	ForwardTo = log.Key("forward_to")
	// Name of the listener in the config file
	Name = log.Key("name")
	// ConfigFile is the filename of the config
	ConfigFile = log.Key("config_file")
	// ReadLen is bytes read from a connection
	ReadLen = log.Key("read_len")
	// ContentLength is the HTTP header content-length value
	ContentLength = log.Key("content_len")
	// MetricType is the type of signalfx metric
	MetricType = log.Key("metric_type")
	// ListenFrom is the listening config
	ListenFrom = log.Key("listen_from")
	// WavefrontLine is a direct line received from wavefront protocol
	WavefrontLine = log.Key("wavefront_line")
)
