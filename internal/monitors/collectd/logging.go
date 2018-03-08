package collectd

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

var logRE = regexp.MustCompile(
	`(?s)` + // Allow . to match newlines
		`\[(?P<timestamp>.*?)\] ` +
		`(?:\[(?P<level>\w+?)\] )?` +
		`(?P<message>(?:(?P<plugin>[\w-]+?): )?.*)`)

func logScanner(output io.ReadCloser) *bufio.Scanner {
	s := bufio.NewScanner(output)
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		lines := bytes.Split(data, []byte{'\n'})
		// If there is no newline in the data, lines will only have one element,
		// so return and wait for more data.
		if len(lines) == 1 && !atEOF {
			return 0, nil, nil
		}

		// For any subsequent indented lines, assume they are part of the same
		// log entry.  This requires that the whole entry be fed to this
		// function in a single chunk, so some entries may get split up
		// erroneously.
		var i int
		for i = 1; i < len(lines) && len(lines[i]) > 0 && (lines[i][0] == ' ' || lines[i][0] == '\t'); i++ {
		}

		entry := bytes.Join(lines[:i], []byte("\n"))
		// the above Join adds back all newlines lost except for one
		return len(entry) + 1, entry, nil
	})
	return s
}

func logLine(line string, logger *log.Entry) {
	groups := utils.RegexpGroupMap(logRE, line)

	var level string
	var message string
	if groups == nil {
		level = "info"
		message = line
	} else {
		if groups["plugin"] != "" {
			logger = logger.WithField("plugin", groups["plugin"])
		}

		level = groups["level"]
		message = strings.TrimPrefix(groups["message"], groups["plugin"]+": ")
	}

	switch level {
	case "debug":
		logger.Debug(message)
	case "info":
		logger.Info(message)
	case "notice":
		logger.Info(message)
	case "warning", "warn":
		logger.Warn(message)
	case "err", "error":
		logger.Error(message)
	default:
		logger.Info(message)
	}
}
