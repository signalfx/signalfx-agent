package spanfilter

import (
	"context"
	"encoding/json"
	"github.com/signalfx/golib/errors"
	"strings"
)

// Map is the response we return from ingest wrt our span endpoint
// It contains the number of spans that were valid, and a map of string reason to spanIds for each invalid span
type Map struct {
	Valid   int                 `json:"valid"`
	Invalid map[string][]string `json:"invalid,omitempty"`
}

var emptySpanFilter = &Map{}

const (
	// OK valid spans
	OK = "ok"
)

//CheckInvalid is a nil safe check if this SpanFilter contains invalid keys
func (s *Map) CheckInvalid() bool {
	return s != nil && len(s.Invalid) > 0
}

// Error returns a json representation of the Map
func (s *Map) Error() string {
	bytes, err := json.Marshal(s)
	resp := "Unable to format Map"
	if err == nil {
		resp = string(bytes)
	}
	return resp
}

// Add a error trace id
func (s *Map) Add(errType string, id string) {
	if strings.ToLower(errType) == OK {
		s.Valid++
	} else {
		if s.Invalid == nil {
			s.Invalid = make(map[string][]string)
		}
		s.Invalid[errType] = append(s.Invalid[errType], id)
	}
}

// FromBytes returns a Map or an error
func FromBytes(body []byte) *Map {
	var spanFilter Map
	if err := json.Unmarshal(body, &spanFilter); err != nil {
		return nil
	}
	return &spanFilter
}

// ReturnInvalidOrError returns nil for a valid SpanFilter, an invalid SpanFilter or an error containing the bytes
func ReturnInvalidOrError(body []byte) error {
	if sf := FromBytes(body); sf != nil {
		if sf.CheckInvalid() {
			return sf
		}
		return nil
	}
	return errors.New(string(body))
}

type streamMetadata int

const (
	spanFailures streamMetadata = iota
)

// WithSpanFilterContext gives you a request with the Map set
func WithSpanFilterContext(ctx context.Context, sf *Map) context.Context {
	return context.WithValue(ctx, spanFailures, sf)
}

// GetSpanFilterMapOrNew is a target for spanumsink.SinkFunc to be turned into a spanumsink.Sink
func GetSpanFilterMapOrNew(ctx context.Context) (context.Context, *Map) {
	v := ctx.Value(spanFailures)
	if v != nil {
		failMap := v.(*Map)
		return ctx, failMap
	}
	failMap := &Map{}
	return context.WithValue(ctx, spanFailures, failMap), failMap
}

// GetSpanFilterMapFromContext is a target for spanumsink.SinkFunc to be turned into a spanumsink.Sink
func GetSpanFilterMapFromContext(ctx context.Context) error {
	v := ctx.Value(spanFailures)
	if v != nil {
		failMap := v.(*Map)
		return failMap
	}
	return emptySpanFilter
}

// IsMap returns whether an error is an instance of Map
func IsMap(err error) bool {
	if err != nil {
		switch err.(type) {
		case *Map:
			return true
		}
	}
	return false
}

// IsInvalid returns false if it's a Map with no invalid entries or nil, else true
func IsInvalid(err error) bool {
	if err != nil {
		switch err.(type) {
		case *Map:
			return err.(*Map).CheckInvalid()
		}
	}
	return err != nil
}
