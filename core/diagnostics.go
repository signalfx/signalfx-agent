package core

import (
	"fmt"
	"net"
	"os"

	yaml "gopkg.in/yaml.v2"

	au "github.com/logrusorgru/aurora"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

const diagnosticSocketPath = "/var/run/signalfx.sock"

// Serves the diagnostic status on the domain socket
func (a *Agent) serveDiagnosticInfo() error {
	os.Remove(diagnosticSocketPath)
	sock, err := net.Listen("unix", diagnosticSocketPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not bind to diagnostic endpoint")
		return err
	}

	go func() {
		for {
			conn, err := sock.Accept()
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Problem accepting diagnostic socket request")
				continue
			}

			_, err = conn.Write([]byte(a.DiagnosticText()))
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Could not write diagnostic information")
			}
			conn.Close()
		}
	}()
	return nil
}

// DiagnosticText returns a simple textual output of the agent's status
func (a *Agent) DiagnosticText() string {
	return fmt.Sprintf(
		au.Bold("NeoAgent Status").String()+
			"\n===============\n"+
			au.Bold("\nAgent Configuration:").String()+
			"\n%s\n\n"+
			"%s\n"+
			"%s\n"+
			"%s",
		utils.IndentLines(configAsDiagnosticText(a.lastConfig), 2),
		a.writer.DiagnosticText(),
		a.observers.DiagnosticText(),
		a.monitors.DiagnosticText())

}

// Converts the config structure to a human friendly output
func configAsDiagnosticText(conf *config.Config) string {
	s, _ := yaml.Marshal(conf)
	return string(s)
}
