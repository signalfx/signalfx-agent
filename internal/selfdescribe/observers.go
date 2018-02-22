package selfdescribe

import (
	"reflect"

	"github.com/signalfx/signalfx-agent/internal/observers"
)

func observersStructMetadata() map[string]structMetadata {
	sms := map[string]structMetadata{}
	for k := range observers.ConfigTemplates {
		t := reflect.TypeOf(observers.ConfigTemplates[k]).Elem()
		sms[k] = getStructMetadata(t)
	}
	return sms
}
