// +build linux

package postgresql

// AUTOGENERATED BY scripts/collectd-template-to-go.  DO NOT EDIT!!

import (
	"text/template"

	"github.com/signalfx/signalfx-agent/pkg/monitors/collectd"
)

// CollectdTemplate is a template for a postgresql collectd config file
var CollectdTemplate = template.Must(collectd.InjectTemplateFuncs(template.New("postgresql")).Parse(`
<LoadPlugin postgresql>
  Interval {{.IntervalSeconds}}
</LoadPlugin>

<Plugin postgresql>
  <Query custom_deadlocks>
    Statement "SELECT deadlocks as num_deadlocks \
        FROM pg_stat_database \
        WHERE datname = $1;"
    Param database
    <Result>
      Type "pg_xact"
      InstancePrefix "num_deadlocks"
      ValuesFrom "num_deadlocks"
    </Result>
  </Query>
  {{range $q := .Queries}}
  <Query "{{$q.Name}}">
    Statement "{{$q.Statement}}"
    {{range $param := $q.Params -}}
    Param "{{$param}}"
    {{- end}}
    {{with $q.PluginInstanceFrom}}PluginInstanceFrom "{{.}}"{{- end}}
    {{with $q.MinVersion}}MinVersion {{.}}{{- end}}
    {{with $q.MaxVersion}}MaxVersion {{.}}{{- end}}
    {{range $r := $q.Results -}}
    <Result>
      Type "{{$r.Type}}"
      {{with $r.InstancePrefix -}}InstancePrefix "{{.}}"{{- end}}
      {{if $r.InstancesFrom -}}InstancesFrom {{range $from := $r.InstancesFrom}}"{{$from}}" {{end}}{{- end}}
      ValuesFrom {{range $v := $r.ValuesFrom}}"{{$v}}" {{- end}}
    </Result>
    {{- end}}
  </Query>
  {{end}}
  {{range $db := .Databases}}
  <Database "{{$db.Name}}">
    Host "{{$.Host}}"
    Port "{{$.Port}}"
    ReportHost {{toBool $.ReportHost}}
    {{if $db.Username -}}User "{{$db.Username}}"{{else if $.Username}}User "{{$.Username}}"{{- end}}
    {{if $db.Password -}}Password "{{$db.Password}}"{{else if $.Password}}Password "{{$.Password}}"{{- end}}
    Instance "{{$db.Name}}[monitorID={{$.MonitorID}}]"
    {{with $db.Interval -}}Interval {{.}}{{- end}}
    {{with $db.ExpireDelay -}}ExpireDelay {{.}}{{- end}}
    {{with $db.SSLMode -}}SSLMode "{{.}}"{{- end}}
    {{with $db.KRBSrvName -}}KRBSrvName "{{.}}"{{- end}}
    {{if $db.Queries}}
    {{range $q := $db.Queries -}}
    Query "{{$q}}"
    {{end}}
    {{- else}}
    Query custom_deadlocks
    Query backends
    Query transactions
    Query queries
    Query queries_by_table
    Query query_plans
    Query table_states
    Query query_plans_by_table
    Query table_states_by_table
    Query disk_io
    Query disk_io_by_table
    Query disk_usage
    {{end}}
  </Database>
  {{end}}
  DefaultQueryConfigPath "{{ bundleDir }}/postgresql_default.conf"
</Plugin>
`)).Option("missingkey=error")
