// Package main contains a simple wrapper for the Fake K8s API Server to run it
// standalone
package main

import (
	"flag"

	"github.com/signalfx/signalfx-agent/pkg/neotest/k8s/testhelpers/fakek8s"
)

func main() {
	flag.Parse()
	server := fakek8s.NewFakeK8s()
	server.Start()
	print("Running test server on " + server.URL())
	select {}
}
