package host

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/observers"
	"github.com/stretchr/testify/require"
)

var (
	exe, _ = os.Executable()
)

func openTestTCPPorts(t *testing.T) []*net.TCPListener {
	tcpLocalhost, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	require.Nil(t, err)

	tcpV6Localhost, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("[::]"),
		Port: 0,
	})
	require.Nil(t, err)

	tcpAllPorts, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 0,
	})
	require.Nil(t, err)

	return []*net.TCPListener{
		tcpLocalhost,
		tcpV6Localhost,
		tcpAllPorts,
	}
}

func openTestUDPPorts(t *testing.T) []*net.UDPConn {
	udpLocalhost, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	require.Nil(t, err)

	udpV6Localhost, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("[::]"),
		Port: 0,
	})
	require.Nil(t, err)

	udpAllPorts, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 0,
	})
	require.Nil(t, err)

	return []*net.UDPConn{
		udpLocalhost,
		udpV6Localhost,
		udpAllPorts,
	}
}

var selfPid = os.Getpid()

func Test_HostObserver(t *testing.T) {
	config := &Config{
		PollIntervalSeconds: 1,
	}

	var o *Observer
	var lock sync.Mutex
	var endpoints map[services.ID]services.Endpoint

	startObserver := func() {
		endpoints = make(map[services.ID]services.Endpoint)

		o = &Observer{
			serviceCallbacks: &observers.ServiceCallbacks{
				Added: func(se services.Endpoint) {
					lock.Lock()
					endpoints[se.Core().ID] = se
					lock.Unlock()
				},
				Removed: func(se services.Endpoint) {
					lock.Lock()
					delete(endpoints, se.Core().ID)
					lock.Unlock()
				},
			},
		}
		err := o.Configure(config)
		if err != nil {
			panic("could not setup observer")
		}
	}

	t.Run("Basic connections", func(t *testing.T) {
		tcpConns := openTestTCPPorts(t)
		udpConns := openTestUDPPorts(t)

		startObserver()

		lock.Lock()
		require.True(t, len(endpoints) >= len(tcpConns)+len(udpConns))

		t.Run("TCP ports", func(t *testing.T) {
			for _, conn := range tcpConns {
				host, port, _ := net.SplitHostPort(conn.Addr().String())
				expectedID := fmt.Sprintf("%s-%s-TCP-%d", host, port, selfPid)
				e := endpoints[services.ID(expectedID)].(*services.EndpointCore)
				require.NotNil(t, e)

				portNum, _ := strconv.Atoi(port)
				require.EqualValues(t, e.Port, portNum)
				require.Equal(t, filepath.Base(exe), e.Name)
				require.Equal(t, e.PortType, services.TCP)
				if host[0] == ':' {
					require.Equal(t, e.DerivedFields()["is_ipv6"], true)
				} else {
					require.Equal(t, e.DerivedFields()["is_ipv6"], false)
				}
			}
		})

		t.Run("UDP Ports", func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("skipping test on windows")
			}
			for _, conn := range udpConns {
				host, port, _ := net.SplitHostPort(conn.LocalAddr().String())
				expectedID := fmt.Sprintf("%s-%s-UDP-%d", host, port, selfPid)
				e := endpoints[services.ID(expectedID)].(*services.EndpointCore)
				require.NotNil(t, e)
				portNum, _ := strconv.Atoi(port)
				require.EqualValues(t, e.Port, portNum)
				require.Equal(t, filepath.Base(exe), e.Name)
				require.Equal(t, services.UDP, e.PortType)
				if host[0] == ':' {
					require.Equal(t, e.DerivedFields()["is_ipv6"], true)
				} else {
					require.Equal(t, e.DerivedFields()["is_ipv6"], false)
				}
			}
		})

		lock.Unlock()
		o.Shutdown()
	})
}
