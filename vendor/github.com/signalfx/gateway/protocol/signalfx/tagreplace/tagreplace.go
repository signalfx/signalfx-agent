package tagreplace

import (
	"context"
	"fmt"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/datapoint/dpsink"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/golib/trace"
	"regexp"
	"strings"
)

type sink interface {
	dpsink.Sink
	trace.Sink
}

var _ sink = &TagReplace{}

// TagReplace does regex search and replace in span Names and puts resultant in tags.
// This modifies the objects parsed, so in a concurrent context, you will want to copy the objects sent here first
type TagReplace struct {
	rules []*regexp.Regexp
	next  sink
}

// AddDatapoints is a passthrough
func (t *TagReplace) AddDatapoints(ctx context.Context, points []*datapoint.Datapoint) error {
	return t.next.AddDatapoints(ctx, points)
}

// AddEvents is a passthrough
func (t *TagReplace) AddEvents(ctx context.Context, events []*event.Event) error {
	return t.next.AddEvents(ctx, events)
}

// AddSpans maps all rules to all spans and does the replacements, this can be VERY expensive, and modifies the spans
func (t *TagReplace) AddSpans(ctx context.Context, spans []*trace.Span) error {
	for _, s := range spans {
		if s.Name != nil {
			for _, r := range t.rules {
				nms := r.SubexpNames()
				oldName := *s.Name
				if ms := r.FindStringSubmatch(oldName); ms != nil {
					ims := r.FindStringSubmatchIndex(oldName)

					var newName []string
					var index = 0
					if s.Tags == nil {
						s.Tags = map[string]string{}
					}

					for i := 1; i < len(ms); i++ {
						s.Tags[nms[i]] = ms[i]
						newName = append(newName, oldName[index:ims[i*2]], "{", nms[i], "}")
						index = ims[i*2+1]
					}
					if index < len(oldName) {
						newName = append(newName, oldName[index:])
					}
					s.Name = pointer.String(strings.Join(newName, ""))
				}
			}
		}
	}
	return t.next.AddSpans(ctx, spans)
}

// New returns you a new TagReplace object
func New(ruleStrings []string, next sink) (*TagReplace, error) {
	var rules []*regexp.Regexp
	for _, r := range ruleStrings {
		var err error
		var rp *regexp.Regexp
		if rp, err = regexp.Compile(r); err != nil {
			return nil, err
		}
		if len(rp.SubexpNames()) < 2 {
			return nil, fmt.Errorf("regex contains no named parenthesized subexpressions '%s'", r)
		}
		for i := 1; i < len(rp.SubexpNames()); i++ {
			v := rp.SubexpNames()[i]
			if len(v) < 1 {
				return nil, fmt.Errorf("regex contains a non named parenthesized subexpression '%s'", r)
			}
		}
		rules = append(rules, rp)
	}
	return &TagReplace{
		rules: rules,
		next:  next,
	}, nil
}
