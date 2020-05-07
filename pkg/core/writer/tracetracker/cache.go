package tracetracker

import (
	"container/list"
	"sync"
	"time"
)

type cacheKey struct {
	dimName  string
	dimValue string
	value    string
}

type cacheElem struct {
	LastSeen time.Time
	Obj      *cacheKey
}

type TimeoutCache struct {
	sync.Mutex

	// How long to keep sending metrics for a particular service name after it
	// is last seen
	timeout time.Duration
	// A linked list of keys sorted by time last seen
	keysByTime *list.List
	// Which keys are active currently.  The value is an entry in the
	// keysByTime linked list so that it can be quickly accessed and
	// moved to the back of the list.
	keysActive map[cacheKey]*list.Element

	// Internal metrics
	ActiveCount int64
	PurgedCount int64
}

// RunIfKeyDoesNotExist locks and runs the supplied function if the key does not exist.
// Be careful not to perform cache operations inside of this function because they will deadlock
func (t *TimeoutCache) RunIfKeyDoesNotExist(o *cacheKey, fn func()) {
	t.Lock()
	defer t.Unlock()
	if _, ok := t.keysActive[*o]; ok {
		return
	}
	fn()
}

// UpdateOrCreate
func (t *TimeoutCache) UpdateOrCreate(o *cacheKey, now time.Time) (isNew bool) {
	t.Lock()
	defer t.Unlock()
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
		t.ActiveCount++
	}
	return
}

// PurgeOld
func (t *TimeoutCache) PurgeOld(now time.Time, onPurge func(*cacheKey)) {
	t.Lock()
	defer t.Unlock()
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

		t.ActiveCount--
		t.PurgedCount++
	}
}

func (t *TimeoutCache) GetActiveCount() int64 {
	t.Lock()
	defer t.Unlock()
	return t.ActiveCount
}

func (t *TimeoutCache) GetPurgedCount() int64 {
	t.Lock()
	defer t.Unlock()
	return t.PurgedCount
}

func NewTimeoutCache(timeout time.Duration) *TimeoutCache {
	return &TimeoutCache{
		timeout:    timeout,
		keysByTime: list.New(),
		keysActive: make(map[cacheKey]*list.Element),
	}
}
