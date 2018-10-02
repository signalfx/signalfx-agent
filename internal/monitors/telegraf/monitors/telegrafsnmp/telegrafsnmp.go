package telegrafsnmp

import (
	"context"
	"fmt"
	"github.com/ulule/deepcopier"
	"time"

	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	telegrafPlugin "github.com/influxdata/telegraf/plugins/inputs/snmp"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const monitorType = "telegraf/snmp"

// MONITOR(telegraf/snmp): This monitor reports metrics from snmp agents.
// This monitor is based on the Telegraf SNMP plugin.  More information about the Telegraf plugin
// can be found [here](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/snmp).
//
// **NOTE:** This snmp monitor does not currently support MIB look ups because of a dependency on `net-snmp`
// and specifically the commands `snmptranslate` and `snmptable`.
//
// Sample YAML configuration:
//
//```yaml
// monitors:
//  - type: telegraf/snmp
//    agents:
//      - "127.0.0.1:161"
//    version: 2
//    community: "public"
//    fields:
//      - name: "uptime"
//        oid: ".1.3.6.1.2.1.1.3.0"
//```
//
// Using a discovery rule to discover and configure for a specific snmp agent
//```yaml
// monitors:
//  - type: telegraf/snmp
//    discoveryRule: container_name =~ "snmp" && port == 161
//    version: 2
//    community: "public"
//    fields:
//      - name: "uptime"
//        oid: ".1.3.6.1.2.1.1.3.0"
//```
//

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Field represents an SNMP field
type Field struct {
	// Name of the field.  The OID will be used if no value is supplied.
	Name string `yaml:"name"`
	// The OID to fetch.
	Oid string `yaml:"oid"`
	// The sub-identifier to strip off when matching indexes to other fields.
	OidIndexSuffix string `yaml:"oidIndexSuffix"`
	// The index length after the table OID.  The index will be truncated after
	// this length in order to remove length index suffixes or non-fixed values.
	OidIndexLength int `yaml:"oidIndexLength"`
	// Whether to output the field as a tag.
	IsTag bool `yaml:"isTag"`
	// Controls the type conversion applied to the value: `"float(X)"`, `"float"`,
	// `"int"`, `"hwaddr"`, `"ipaddr"` or `""` (default).
	Conversion string `yaml:"conversion"`
}

// Table represents an SNMP table
type Table struct {
	// Metric name.  If not supplied the OID will be used.
	Name string `yaml:"name"`
	// Top level tags to inherit.
	InheritTags []string `yaml:"inheritTags"`
	// Add a tag for the table index for each row.
	IndexAsTag bool `yaml:"indexAsTag"`
	// Specifies the ags and values to look up.
	Fields []Field `yaml:"field"`
	// The OID to fetch.
	Oid string `yaml:"oid"`
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"true"`
	// Host and port will be concatenated and appended to the list of SNMP agents to connect to.
	Host                 string `yaml:"host"`
	// Port and Host will be concatenated and appended to the list of SNMP agents to connect to.
	Port                 uint16 `yaml:"port"`
	// SNMP agent address and ports to query for information.  An example address is `0.0.0.0:5555`
	// If an address is supplied with out a port, the default port `161` will be used.
	Agents []string `yaml:"agents"`
	// The number of times to retry.
	Retries int `yaml:"retries"`
	// The SNMP protocol version to use (ie: `1`, `2`, `3`).
	Version uint8
	// The SNMP community to use.
	Community string `yaml:"community" default:"public"`
	// Maximum number of iterations for reqpeating variables
	MaxRepetitions uint8 `yaml:"maxRepetitions" default:"50"`
	// SNMP v3 context name to use with requests
	ContextName string `yaml:"contextName"`
	// Security level to use for SNMP v3 messages: `noAuthNoPriv` `authNoPriv`, `authPriv`.
	SecLevel string `yaml:"secLevel" default:"noAuthNoPriv"`
	// Name to used to authenticate with SNMP v3 requests.
	SecName string `yaml:"secName"`
	// Protocol to used to authenticate SNMP v3 requests: `"MD5"`, `"SHA"`, and `""` (default).
	AuthProtocol string `yaml:"authProtocol" default:""`
	// Password used to authenticate SNMP v3 requests.
	AuthPassword string `yaml:"authPassword" default:"" neverLog:"true"`
	// Protocol used for encrypted SNMP v3 messages: `DES`, `AES`, `""` (default).
	PrivProtocol string `yaml:"privProtocol" default:""`
	// Password used to encrypt SNMP v3 messages.
	PrivPassword string `yaml:"privPassword"`
	// The SNMP v3 engine ID.
	EngineID string `yaml:"engineID"`
	// The SNMP v3 engine boots.
	EngineBoots uint32 `yaml:"engineBoots"`
	// The SNMP v3 engine time.
	EngineTime uint32 `yaml:"engineTime"`
	// The top-level measurement name
	Name string `yaml:"name"`
	// The top-level SNMP fields
	Fields []Field `yaml:"fields"`
	// SNMP Tables
	Tables []Table `yaml:"tables"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
}

// fetch the factory used to generate the perf counter plugin
var factory = telegrafInputs.Inputs["snmp"]

// converts our config struct for field to a telegraf field
func getTelegrafFields(incoming []Field) []telegrafPlugin.Field{
	// initialize telegraf fields
	fields := make([]telegrafPlugin.Field, 0, len(incoming))

	// copy fields to table
	for _, field := range incoming {
		f := telegrafPlugin.Field{}
		deepcopier.Copy(&field).To(&f)
		fields = append(fields, f)
	}

	return fields
}

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) (err error) {
	plugin := factory().(*telegrafPlugin.Snmp)

	// create the emitter
	em := baseemitter.NewEmitter(m.Output, logger)

	// set a default plugin dimension
	em.AddTag("plugin", "snmp")

	// create the accumulator
	ac := accumulator.NewAccumulator(em)

	// copy configurations to the plugin
	if err = deepcopier.Copy(conf).To(plugin); err != nil {
		logger.Error("unable to copy configurations to plugin")
		return err
	}

	// if a service is discovered that exposes snmp, take the host and port and add them to the agents list
	if conf.Host != "" {
		if plugin.Agents == nil {
			plugin.Agents = []string{fmt.Sprintf("%s:%d", conf.Host, conf.Port)}
		} else {
			plugin.Agents = append(plugin.Agents, fmt.Sprintf("%s:%d", conf.Host, conf.Port))
		}
	}

	// get top level telegraf fields
	plugin.Fields = getTelegrafFields(conf.Fields)

	// initialize plugin.Tables
	plugin.Tables = make([]telegrafPlugin.Table, 0, len(conf.Tables))

	// copy tables
	for _, table := range conf.Tables {
		t := telegrafPlugin.Table{}
		deepcopier.Copy(&table).To(&t)

		// get telegraf fields
		t.Fields = getTelegrafFields(table.Fields)

		plugin.Tables = append(plugin.Tables, t)
	}

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return err
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
