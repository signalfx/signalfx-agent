package correlations

import (
	"net/http"

	"github.com/golang/groupcache/lru"
)

// deduplicator deduplicates requests and cancels pending conflicting requests and deduplicates
// this is not threadsafe
type deduplicator struct {
	// maps for deduplicating requests
	pendingCreates *lru.Cache
	pendingDeletes *lru.Cache
}

func (d *deduplicator) dedupCorrelate(r *request) bool {
	// look for duplicate pending creates
	pendingCreate, ok := d.pendingCreates.Get(*r.Correlation)
	if ok && pendingCreate.(*contextWithCancel).ctx.Err() == nil {
		// return true if there is a context for the key and the context has not expired
		return true
	}

	// store the new request's context cancellation function
	d.pendingCreates.Add(*r.Correlation, r.contextWithCancel)

	// cancel any pending delete operations
	cancelDelete, deletePending := d.pendingDeletes.Get(*r.Correlation)
	if deletePending {
		cancelDelete.(*contextWithCancel).cancel()
		d.pendingDeletes.Remove(*r.Correlation)
	}

	return false
}

func (d *deduplicator) dedupDelete(r *request) bool {
	// look for duplicate deletes
	pendingCreate, ok := d.pendingDeletes.Get(*r.Correlation)
	if ok && pendingCreate.(*contextWithCancel).ctx.Err() == nil {
		return true
	}

	// store the new request's context cancellation function
	d.pendingDeletes.Add(*r.Correlation, r.contextWithCancel)

	// cancel any pending create operations
	cancelCreate, createPending := d.pendingCreates.Get(*r.Correlation)
	if createPending {
		cancelCreate.(*contextWithCancel).cancel()
		d.pendingCreates.Remove(*r.Correlation)
	}

	return false
}

// isDup returns true if the request is a duplicate
func (d *deduplicator) isDup(r *request) (isDup bool) {
	switch r.operation {
	case http.MethodPut:
		return d.dedupCorrelate(r)
	case http.MethodDelete:
		return d.dedupDelete(r)
	default:
		return
	}
}

// newDeduplicator returns a new instance
func newDeduplicator(size int) *deduplicator {
	return &deduplicator{
		pendingCreates: lru.New(size),
		pendingDeletes: lru.New(size),
	}
}
