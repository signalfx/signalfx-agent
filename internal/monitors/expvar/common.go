package expvar

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/signalfx/golib/v3/datapoint"
)

var capRegexp = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func snakeCaseSlice(slice []string) []string {
	var words []string
	for _, s := range slice {
		var capWords []string
		for _, matchedCapWords := range capRegexp.FindAllStringSubmatch(s, -1) {
			for _, matchedCapWord := range matchedCapWords[1:] {
				if matchedCapWord = strings.TrimSpace(matchedCapWord); matchedCapWord != "" {
					capWords = append(capWords, matchedCapWord)
				}
			}
		}
		words = append(words, strings.ToLower(strings.Join(capWords, "_")))
	}
	return words
}

var wordRegexp = regexp.MustCompile("[a-zA-Z]+")

func joinWords(slice []string, sep string) string {
	var words []string
	for _, s := range slice {
		if wordRegexp.MatchString(s) {
			words = append(words, s)
		}
	}
	return strings.Join(words, sep)
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
