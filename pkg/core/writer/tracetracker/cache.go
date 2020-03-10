package tracetracker

import (
	"container/list"
	"sync/atomic"
	"time"
)

type CacheKey struct {
	dimName     string
	dimValue    string
	environment string
	service     string
}

type cacheElem struct {
	LastSeen time.Time
	Obj      *CacheKey
}

type TimeoutCache struct {
	// How long to keep sending metrics for a particular service name after it
	// is last seen
	timeout time.Duration
	// A linked list of keys sorted by time last seen
	keysByTime *list.List
	// Which keys are active currently.  The value is an entry in the
	// keysByTime linked list so that it can be quickly accessed and
	// moved to the back of the list.
	keysActive map[CacheKey]*list.Element

	// Internal metrics
	ActiveCount int64
	PurgedCount int64
}

// UpdateOrCreate
func (t *TimeoutCache) UpdateOrCreate(o *CacheKey, now time.Time) (isNew bool) {
	if timeElm, ok := t.keysActive[*o]; ok {
		timeElm.Value.(*cacheElem).LastSeen = now
		t.keysByTime.MoveToFront(timeElm)
	} else {
		isNew = true
		elm := t.keysByTime.PushFront(&cacheElem{
			LastSeen: now,
			Obj:      o,
		})
		t.keysActive[*o] = elm
		atomic.AddInt64(&t.ActiveCount, 1)
	}
	return
}

// PurgeOld
func (t *TimeoutCache) PurgeOld(now time.Time, onPurge func(*CacheKey)) {
	for {
		elm := t.keysByTime.Back()
		if elm == nil {
			break
		}
		e := elm.Value.(*cacheElem)
		// If this one isn't timed out, nothing else in the list is either.
		if now.Sub(e.LastSeen) < t.timeout {
			break
		}

		t.keysByTime.Remove(elm)
		delete(t.keysActive, *e.Obj)
		onPurge(e.Obj)

		atomic.AddInt64(&t.ActiveCount, -1)
		atomic.AddInt64(&t.PurgedCount, 1)
	}
}

func NewTimeoutCache(timeout time.Duration) *TimeoutCache {
	return &TimeoutCache{
		timeout:    timeout,
		keysByTime: list.New(),
		keysActive: make(map[CacheKey]*list.Element),
	}
}
