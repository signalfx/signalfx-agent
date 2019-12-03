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

	inBraces := false
	currentField := ""
	for i, c := range p {
		switch c {
		case '{':
			if inBraces {
				logger.Errorf("Invalid pattern, cannot nest opening braces '{' in pattern '%s'", p)
				return nil
			}
			inBraces = true
			if len(currentField) > 0 {
				substrs = append(substrs, currentField)
			} else if i != 0 {
				logger.Errorf("Cannot have back to back match groups in pattern '%s'", p)
				return nil
			}
			currentField = ""
		case '}':
			if !inBraces {
				logger.Errorf("Invalid pattern, no opening '{' found for pattern '%s'", p)
				return nil
			}
			inBraces = false
			substrs = append(substrs, currentField)
			currentField = ""
		default:
			currentField += string(c)
		}
	}

	if inBraces {
		logger.Errorf("Invalid pattern, no ending } found for pattern '%s'", p)
		return nil
	}

	if len(currentField) > 0 {
		substrs = append(substrs, currentField)
	}

	return &fieldPattern{
		substrs:        substrs,
		startWithField: strings.HasPrefix(p, "{"),
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
