package expvar

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/signalfx/signalfx-agent/internal/utils"

	"github.com/signalfx/golib/datapoint"
)

var camelRegexp = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func toSnakeCase(s string, sep rune, escape rune) string {
	snake := ""
	splits, _ := utils.SplitString(s, sep, escape)
	for _, split := range splits {
		for _, submatches := range camelRegexp.FindAllStringSubmatch(split, -1) {
			for _, submatch := range submatches[1:] {
				submatch = strings.TrimSpace(submatch)
				if submatch != "" {
					snake += submatch + "_"
				}
			}
		}
		snake = strings.TrimSuffix(strings.TrimSuffix(snake, "."), "_") + "."
	}
	return strings.ToLower(strings.TrimSuffix(snake, "."))
}

// getMostRecentGCPauseIndex logic is derived from https://golang.org/pkg/runtime/ in the PauseNs section of the 'type MemStats' section
func getMostRecentGCPauseIndex(dpsMap map[string][]*datapoint.Datapoint) int64 {
	dps := dpsMap[memstatsNumGCMetricPath]
	mostRecentGCPauseIndex := int64(-1)
	if len(dps) > 0 && dps[0].Value != nil {
		if numGC, err := strconv.ParseInt(dps[0].Value.String(), 10, 0); err == nil {
			mostRecentGCPauseIndex = (numGC + 255) % 256
		}
	}
	return mostRecentGCPauseIndex
}

var slashLastRegexp = regexp.MustCompile(`[^\/]*$`)

func getApplicationName(values map[string]interface{}) (string, error) {
	if cmdline, ok := values["cmdline"].([]interface{}); ok && len(cmdline) > 0 {
		name, ok := cmdline[0].(string)
		if !ok {
			return "", errors.New("unable to obtain app name")
		}
		if applicationName := strings.TrimSpace(slashLastRegexp.FindStringSubmatch(name)[0]); applicationName != "" {
			return applicationName, nil
		}
	}
	return "", errors.New("cmdline map not defined")
}
