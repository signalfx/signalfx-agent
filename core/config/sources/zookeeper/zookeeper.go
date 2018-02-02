// Package zookeeper contains the logic for using Zookeeper as a config source.
// Currently globbing only works if it is a suffix to the path.
package zookeeper

import (
	"fmt"
	"hash/crc64"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/creasty/defaults"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/signalfx/neo-agent/core/config/sources/types"
	log "github.com/sirupsen/logrus"
)

type zkConfigSource struct {
	conn      *zk.Conn
	endpoints []string
	timeout   time.Duration
	table     *crc64.Table
}

// Config is used to configure the Zookeeper client
type Config struct {
	Endpoints      []string `yaml:"endpoints"`
	TimeoutSeconds uint     `yaml:"timeoutSeconds" default:"10"`
}

// New creates a new Zookeeper config source with the given config.  All gets
// and watches will use the same client.
func New(conf *Config) types.ConfigSource {
	defaults.Set(conf)
	return &zkConfigSource{
		endpoints: conf.Endpoints,
		table:     crc64.MakeTable(crc64.ECMA),
		timeout:   time.Duration(conf.TimeoutSeconds) * time.Second,
	}
}

func (z *zkConfigSource) ensureConnection() error {
	if z.conn != nil && z.conn.State() != zk.StateDisconnected {
		return nil
	}

	conn, _, err := zk.Connect(z.endpoints, z.timeout)
	if err != nil {
		return err
	}
	z.conn = conn
	return nil
}

func isGlob(path string) (string, bool) {
	if strings.HasSuffix(path, "/*") {
		return strings.TrimSuffix(path, "/*"), true
	}
	return "", false
}

func (z *zkConfigSource) Name() string {
	return "zk"
}

func (z *zkConfigSource) Get(path string) (map[string][]byte, uint64, error) {
	content, version, _, err := z.getNodes(path, false)
	return content, version, err
}

// The Zookeeper go lib is really not amenable to the pattern we use for
// ConfigSource.
func (z *zkConfigSource) getNodes(path string, watch bool) (map[string][]byte, uint64, []<-chan zk.Event, error) {
	if err := z.ensureConnection(); err != nil {
		return nil, 0, nil, err
	}

	contentMap := make(map[string][]byte)
	var sums string
	var nodes []string
	var events []<-chan zk.Event

	prefix, globbed := isGlob(path)
	if globbed {
		var err error
		var parentEvents <-chan zk.Event
		if watch {
			nodes, _, parentEvents, err = z.conn.ChildrenW(prefix)
		} else {
			nodes, _, err = z.conn.Children(prefix)
		}

		if err != nil {
			return nil, 0, nil, err
		}
		if parentEvents != nil {
			events = append(events, parentEvents)
		}
	} else {
		nodes = []string{path}
	}

	sort.Strings(nodes)

	for _, n := range nodes {
		var content []byte
		var stat *zk.Stat
		var err error
		var nodeEvents <-chan zk.Event

		fullPath := filepath.Join(prefix, n)

		if watch {
			content, stat, nodeEvents, err = z.conn.GetW(fullPath)
		} else {
			content, stat, err = z.conn.Get(fullPath)
		}
		if err != nil {
			return nil, 0, nil, err
		}

		contentMap[n] = content
		sums = fmt.Sprintf("%s:%s:%d", sums, fullPath, stat.Version)

		if nodeEvents != nil {
			events = append(events, nodeEvents)
		}
	}

	return contentMap, crc64.Checksum([]byte(sums), z.table), events, nil
}

func (z *zkConfigSource) WaitForChange(path string, version uint64, stop <-chan struct{}) error {
	_, newVersion, events, err := z.getNodes(path, true)
	if err != nil {
		return err
	}

	if version != newVersion {
		return nil
	}

	cases := []reflect.SelectCase{
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(stop),
		},
	}
	for _, ch := range events {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}

	for {
		log.Debugf("Waiting for ZK change at %s with %d nodes", path, len(cases)-1)
		chosen, _, _ := reflect.Select(cases)
		// Stop channel is the first
		if chosen == 0 {
			return nil
		}
		log.Debugf("ZK path %s changed", path)

		// Get the data again and compare versions so that we don't have false
		// positives.
		_, newVersion, err := z.Get(path)
		if err != nil {
			return err
		}
		if newVersion != version {
			return nil
		}
	}
}
