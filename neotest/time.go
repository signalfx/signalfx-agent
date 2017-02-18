package neotest

import "time"

// FixedTime is always at 2016-01-05T15:04:05-06:00
var FixedTime func() time.Time

func init() {
	if fixedTime, err := time.Parse(time.RFC3339, "2016-01-05T15:04:05-06:00"); err != nil {
		panic("unable to parse time")
	} else {
		FixedTime = func() time.Time { return fixedTime }
	}
}
