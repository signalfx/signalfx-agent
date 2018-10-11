package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/shirou/gopsutil/net"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/observers"
	"github.com/stretchr/testify/assert"
)

var basicConnectionStatJSON = fmt.Sprintf(`[
 {"fd":3,"family":2,"type":1,"localaddr":{"ip":"127.9.8.7","port":14839},"remoteaddr":{"ip":"0.0.0.0","port":0},"status":"LISTEN","uids":[0,0,0,0],"pid":12780}
,{"fd":7,"family":2,"type":1,"localaddr":{"ip":"127.9.8.7","port":14839},"remoteaddr":{"ip":"127.0.0.1","port":55128},"status":"ESTABLISHED","uids":[0,0,0,0],"pid":12780}
,{"fd":0,"family":2,"type":1,"localaddr":{"ip":"127.9.8.7","port":14839},"remoteaddr":{"ip":"127.0.0.1","port":55264},"status":"TIME_WAIT","uids":[],"pid":0} 
,{"fd":0,"family":2,"type":1,"localaddr":{"ip":"127.9.8.7","port":14839},"remoteaddr":{"ip":"127.0.0.1","port":55266},"status":"TIME_WAIT","uids":[],"pid":0} 
,{"fd":7,"family":2,"type":1,"localaddr":{"ip":"127.0.0.1","port":55128},"remoteaddr":{"ip":"127.9.8.7","port":14839},"status":"ESTABLISHED","uids":[0,0,0,0],"pid":12793}
,{"fd":3,"family":2,"type":2,"localaddr":{"ip":"5.4.3.2","port":80},"remoteaddr":{"ip":"0.0.0.0","port":0},"status":"LISTEN","uids":[0,0,0,0],"pid":12780}
,{"fd":3,"family":2,"type":1,"localaddr":{"ip":"10.2.3.4","port":9001},"remoteaddr":{"ip":"0.0.0.0","port":0},"status":"LISTEN","uids":[0,0,0,0],"pid":12768}
,{"fd":3,"family":%d,"type":1,"localaddr":{"ip":"::","port":9000},"remoteaddr":{"ip":"::","port":0},"status":"LISTEN","uids":[0,0,0,0],"pid":12768}
,{"fd":5,"family":1,"type":1,"localaddr":{"ip":"/var/run/signalfx.sock","port":0},"remoteaddr":{"ip":"","port":0},"status":"NONE","uids":[0,0,0,0],"pid":12780 }
,{"fd":6,"family":1,"type":1,"localaddr":{"ip":"/var/run/signalfx-agent-metrics.sock","port":0},"remoteaddr":{"ip":"","port":0},"status":"NONE","uids":[0,0,0,0],"pid":12780}
,{"fd":11,"family":1,"type":1,"localaddr":{"ip":"","port":0},"remoteaddr":{"ip":"","port":0},"status":"NONE","uids":[0,0,0,0],"pid":12793}
,{"fd":0,"family":1,"type":1,"localaddr":{"ip":"/var/run/docker.sock","port":0},"remoteaddr":{"ip":"","port":0},"status":"NONE","uids":[],"pid":0}
]`, syscall.AF_INET6)

func Test_HostObserver(t *testing.T) {
	config := &Config{
		PollIntervalSeconds: 1,
	}

	var o *Observer
	var endpoints map[services.ID]services.Endpoint

	setup := func(connectionStatJSON string, processNameMap map[int32]string) {
		endpoints = make(map[services.ID]services.Endpoint)

		o = &Observer{
			serviceCallbacks: &observers.ServiceCallbacks{
				Added:   func(se services.Endpoint) { endpoints[se.Core().ID] = se },
				Removed: func(se services.Endpoint) { delete(endpoints, se.Core().ID) },
			},
			hostInfoProvider: &fakeHostInfoProvider{
				connectionStats: parseConnectionStatJSON(connectionStatJSON),
				processNameMap:  processNameMap,
			},
		}
		o.Configure(config)
	}

	t.Run("Basic connections", func(t *testing.T) {
		setup(basicConnectionStatJSON, map[int32]string{12780: "agent", 12768: "service"})

		assert.Len(t, endpoints, 3)

		t.Run("IPV4 Port", func(t *testing.T) {
			e := endpoints["10.2.3.4-9001-12768"].(*services.EndpointCore)
			assert.NotNil(t, e)
			assert.EqualValues(t, e.Port, 9001)
			assert.Equal(t, e.Name, "service")
			assert.Equal(t, e.PortType, services.TCP)
		})

		t.Run("IPV4 UDP Port", func(t *testing.T) {
			e := endpoints["5.4.3.2-80-12780"].(*services.EndpointCore)
			assert.NotNil(t, e)
			assert.EqualValues(t, e.Port, 80)
			assert.Equal(t, e.Name, "agent")
			assert.Equal(t, e.PortType, services.UDP)
		})

		t.Run("IPV6 Port", func(t *testing.T) {
			e, keyExists := endpoints["::-9000-12768"] //.(*services.EndpointCore)
			assert.False(t, keyExists)
			assert.Nil(t, e)
		})
	})

	t.Run("PID missing (due to race)", func(t *testing.T) {
		setup(basicConnectionStatJSON, map[int32]string{12768: "service"})

		assert.Len(t, endpoints, 1)
	})

	t.Run("No connections", func(t *testing.T) {
		setup("[]", map[int32]string{})

		assert.Len(t, endpoints, 0)
	})
}

func parseConnectionStatJSON(jsonStr string) []net.ConnectionStat {
	var res []net.ConnectionStat
	err := json.Unmarshal([]byte(jsonStr), &res)
	if err != nil {
		panic("Test JSON is invalid " + err.Error())
	}
	return res
}

type fakeHostInfoProvider struct {
	connectionStats []net.ConnectionStat
	processNameMap  map[int32]string
}

func (f *fakeHostInfoProvider) AllConnectionStats() ([]net.ConnectionStat, error) {
	return f.connectionStats, nil
}

func (f *fakeHostInfoProvider) ProcessNameFromPID(pid int32) (string, error) {
	name, ok := f.processNameMap[pid]

	if !ok {
		return "", errors.New("no process name in map")
	}

	return name, nil
}
