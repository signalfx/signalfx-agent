package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"flag"

	"github.com/signalfx/neo-agent/plugins/monitors/collectd/config"
)

func main() {
	var yamlConfig, outputFile string

	flag.StringVar(&yamlConfig, "c", "", "yaml configuration")
	flag.StringVar(&outputFile, "o", "", "output file")
	flag.Parse()

	// Add command line argument.
	appConfig, err := config.LoadYamlConfig(yamlConfig)
	if err != nil {
		fmt.Printf("Loading yaml failed: %s\n", err)
		os.Exit(1)
	}
	if output, err := config.RenderCollectdConf("etc/collectd/templates", appConfig); err != nil {
		fmt.Printf("Rendering collectd failed: %s\n", err)
		os.Exit(1)
	} else {
		if len(outputFile) > 0 {
			if err := ioutil.WriteFile(outputFile, []byte(output), 0644); err != nil {
				panic(err)
			}
		} else {
			println(output)
		}
	}

	fmt.Printf("%+v\n", appConfig.AgentConfig)
	fmt.Printf("%+v\n", appConfig.Plugins)

	for _, p := range appConfig.Plugins {
		println("plugin")
		for _, tmp := range p.GetTemplates() {
			fmt.Printf("   %s\n", tmp)
		}
	}
}
