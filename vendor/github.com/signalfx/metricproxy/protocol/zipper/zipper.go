package zipper

import (
	"bytes"
	"compress/gzip"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/log"
	"github.com/signalfx/golib/sfxclient"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

// ReadZipper creates a Pool that contains previously used Readers and can create new ones if we run out.
type ReadZipper struct {
	zippers   sync.Pool
	Log       log.Logger
	NewCount  int64
	HitCount  int64
	MissCount int64
	ErrCount  int64
}

// GzipHandler transparently decodes your possibly gzipped request
func (z *ReadZipper) GzipHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		if "gzip" == r.Header.Get("Content-Encoding") {
			gzi := z.zippers.Get()
			if gzi != nil {
				gz := gzi.(*gzip.Reader)
				// put it back
				defer z.zippers.Put(gz)
				err = gz.Reset(r.Body)
				if err == nil {
					defer log.IfErr(z.Log, gz.Close())
					// nasty? could construct another object but seems expensive
					r.Body = gz
					atomic.AddInt64(&z.HitCount, 1)
					h.ServeHTTP(w, r)
					return
				}
			}
		}
		if err != nil {
			atomic.AddInt64(&z.ErrCount, 1)
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte("error handling gzip compressed request " + err.Error()))
			log.IfErr(z.Log, err)
			return
		}
		atomic.AddInt64(&z.MissCount, 1)
		h.ServeHTTP(w, r)
	})
}

// Datapoints implements Collector interface and returns metrics
func (z *ReadZipper) Datapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.CumulativeP("zipper.hitCount", nil, &z.HitCount),
		sfxclient.CumulativeP("zipper.missCount", nil, &z.MissCount),
		sfxclient.CumulativeP("zipper.newCount", nil, &z.NewCount),
		sfxclient.CumulativeP("zipper.errCount", nil, &z.ErrCount),
	}
}

// NewZipper gives you a ReadZipper
func NewZipper() *ReadZipper {
	return newZipper(gzip.NewReader)
}

func newZipper(zipperFunc func(r io.Reader) (*gzip.Reader, error)) *ReadZipper {
	z := &ReadZipper{}
	z.zippers = sync.Pool{New: func() interface{} {
		atomic.AddInt64(&z.NewCount, 1)
		// This is just the header of an empty gzip, unlike NewWriter, i can't pass in nil ot empty bytes
		g, err := zipperFunc(bytes.NewBuffer([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255}))
		if err == nil {
			return g
		}
		return nil
	}}
	return z
}
