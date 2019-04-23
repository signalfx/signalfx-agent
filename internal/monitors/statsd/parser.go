package statsd

import (
	"strings"
)

type fieldPattern struct {
	substrs        []string
	startWithField bool
}

type converter struct {
	pattern *fieldPattern
	metric  *fieldPattern
}

func initConverter(input *ConverterInput) *converter {
	pattern := parseFields(input.Pattern)
	metric := parseFields(input.MetricName)

	if pattern == nil || metric == nil {
		return nil
	}

	return &converter{
		pattern: parseFields(input.Pattern),
		metric:  parseFields(input.MetricName),
	}
}

// parsePattern takes a pattern string and convert it into parsed fieldPattern object
func parseFields(p string) *fieldPattern {
	var substrs []string
	var startWithField bool

	l := len(p)
	wa := 0
	var wb, wc int

	for wa < l {
		wb = wa + strings.Index(p[wa:], "{")
		wc = wa + strings.Index(p[wa:], "}")

		if wa == 0 && wb == 0 {
			startWithField = true
		}

		if wa != wb {
			// edge case : pattern not ending with { and }
			if wb == -1 {
				logger.Errorf("Invalid pattern. Mismatched brackets : %s", p)
				return nil // Invalid pattern, skip.
			}
			substrs = append(substrs, p[wa:wb])
		}

		if wb != -1 && wc > wb {
			substrs = append(substrs, p[wb+1:wc])
			wa = wc + 1
		} else {
			logger.Errorf("Invalid pattern. Mismatched brackets : %s", p)
			return nil // Invalid pattern, skip.
		}
	}

	return &fieldPattern{
		substrs:        substrs,
		startWithField: startWithField,
	}
}

// convertMetric takes a statsd metric name and a list of fieldPattern objects
// and return the dimensions from the first matching pattern
func convertMetric(name string, converters []*converter) (string, map[string]string) {
	for _, c := range converters {
		fields := make(map[string]string)
		w := 0
		i := 0

		if !c.pattern.startWithField {
			if len(name) < len(c.pattern.substrs[0]) || strings.Compare(name[:len(c.pattern.substrs[0])], c.pattern.substrs[0]) != 0 {
				continue
			}
			w = len(c.pattern.substrs[0])
			i = 1
		}

		var next int
		for i < len(c.pattern.substrs) {
			if i == len(c.pattern.substrs)-1 {
				if len(c.pattern.substrs[i]) > 0 {
					fields[c.pattern.substrs[i]] = name[w:]
				}
				i++
				break
			} else {
				next = strings.Index(name[w:], c.pattern.substrs[i+1])

				// Pattern mismatch, skip.
				if next == -1 {
					break
				}
				if len(c.pattern.substrs[i]) > 0 {
					fields[c.pattern.substrs[i]] = name[w : w+next]
				}
				w = w + next + len(c.pattern.substrs[i+1])
				i += 2
			}
		}

		// Pattern mismatch, skip.
		if i < len(c.pattern.substrs) {
			continue
		}

		// Compose metricName
		var metricName string

		isField := c.metric.startWithField
		for _, substr := range c.metric.substrs {
			if isField {
				if v, exists := fields[substr]; exists {
					metricName += v
				}
			} else {
				metricName += substr
			}
			isField = !isField
		}

		return metricName, fields
	}

	return name, nil
}
