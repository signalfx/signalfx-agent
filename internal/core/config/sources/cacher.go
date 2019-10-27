package sources

import (
	"context"
	"sync"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config/types"
	log "github.com/sirupsen/logrus"
)

type configSourceCacher struct {
	sync.Mutex
	source        types.ConfigSource
	cache         map[string]map[string][]byte
	ctx           context.Context
	shouldWatch   bool
	notifications chan<- string
}

func newConfigSourceCacher(ctx context.Context, source types.ConfigSource, notifications chan<- string, shouldWatch bool) *configSourceCacher {
	return &configSourceCacher{
		source:        source,
		ctx:           ctx,
		shouldWatch:   shouldWatch,
		notifications: notifications,
		cache:         make(map[string]map[string][]byte),
	}
}

// optional controls whether it treats a path not found error as a real error
// that causes watching to never be initiated.
func (c *configSourceCacher) Get(path string) (map[string][]byte, error) {
	c.Lock()
	defer c.Unlock()

	if v, ok := c.cache[path]; ok {
		return v, nil
	}

	sourceIter := NewConfigSourceIterator(c.source, path)

	val, err := sourceIter.Next(c.ctx)
	if err != nil {
		return nil, err
	}

	c.cache[path] = val

	if c.shouldWatch {
		go func() {
			for {
				val, err := sourceIter.Next(c.ctx)
				// We were cancelled on so exit this goroutine
				if c.ctx.Err() != nil {
					return
				}

				if err != nil {
					log.WithFields(log.Fields{
						"path":   path,
						"source": c.source.Name(),
						"error":  err,
					}).Error("Could not get next value from config source")
					time.Sleep(10 * time.Second)
					continue
				}

				c.Lock()
				c.cache[path] = val
				c.Unlock()

				c.notifyChanged(path)
			}
		}()
	}

	return val, nil
}

func (c *configSourceCacher) notifyChanged(path string) {
	c.notifications <- c.source.Name() + ":" + path
}
