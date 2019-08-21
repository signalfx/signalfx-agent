package host

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/observers"
	"github.com/stretchr/testify/require"
)

var (
	exe, _ = os.Executable()
)

func openTestTCPPorts(t *testing.T) []net.Addr {
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

	return []net.Addr{
		tcpLocalhost.Addr(),
		tcpV6Localhost.Addr(),
		tcpAllPorts.Addr(),
	}
}

func openTestUDPPorts(t *testing.T) []net.Addr {
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

	return []net.Addr{
		udpLocalhost.LocalAddr(),
		udpV6Localhost.LocalAddr(),
		udpAllPorts.LocalAddr(),
	}
}

var selfPid = os.Getpid()

func Test_HostObserver(t *testing.T) {
	config := &Config{
		PollIntervalSeconds: 1,
	}

	var o *Observer
	var endpoints map[services.ID]services.Endpoint

	startObserver := func() {
		endpoints = make(map[services.ID]services.Endpoint)

		o = &Observer{
			serviceCallbacks: &observers.ServiceCallbacks{
				Added:   func(se services.Endpoint) { endpoints[se.Core().ID] = se },
				Removed: func(se services.Endpoint) { delete(endpoints, se.Core().ID) },
			},
		}
		err := o.Configure(config)
		if err != nil {
			panic("could not setup observer")
		}
	}

	t.Run("Basic connections", func(t *testing.T) {
		tcpPorts := openTestTCPPorts(t)
		udpPorts := openTestUDPPorts(t)

		startObserver()

		require.True(t, len(endpoints) >= 6)

		t.Run("TCP ports", func(t *testing.T) {
			for _, addr := range tcpPorts {
				host, port, _ := net.SplitHostPort(addr.String())
				expectedID := fmt.Sprintf("%s-%s-TCP-%d", host, port, selfPid)
				e := endpoints[services.ID(expectedID)].(*services.EndpointCore)
				require.NotNil(t, e)

				portNum, _ := strconv.Atoi(port)
				require.EqualValues(t, e.Port, portNum)
				require.Equal(t, filepath.Base(exe), e.Name)
				require.Equal(t, e.PortType, services.TCP)
			}
		})

		t.Run("UDP Ports", func(t *testing.T) {
			for _, addr := range udpPorts {
				host, port, _ := net.SplitHostPort(addr.String())
				expectedID := fmt.Sprintf("%s-%s-UDP-%d", host, port, selfPid)
				e := endpoints[services.ID(expectedID)].(*services.EndpointCore)
				require.NotNil(t, e)
				portNum, _ := strconv.Atoi(port)
				require.EqualValues(t, e.Port, portNum)
				require.Equal(t, filepath.Base(exe), e.Name)
				require.Equal(t, services.UDP, e.PortType)
			}
		})
	})
}
