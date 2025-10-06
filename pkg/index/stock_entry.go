package index

import (
	"sync/atomic"

	"github.com/matst80/slask-finder/pkg/types"
)

// stockEntry is a lock-free container for a set of item IDs (types.ItemList).
// It uses an atomic pointer to an immutable map. Each mutation (add/remove)
// builds a fresh copy and attempts a CAS. Readers obtain a snapshot without locks.
//
// Rationale:
// - Avoids per-entry mutex contention when there are many readers.
// - Writers pay a copy cost proportional to the size of the set (expected small for stock locations).
// - Readers see a consistent view (the map referenced by the atomic pointer never mutates).
type stockEntry struct {
	data atomic.Pointer[types.ItemList]
}

// newStockEntry creates a stockEntry with an empty immutable set.
func newStockEntry() *stockEntry {
	se := &stockEntry{}
	empty := make(types.ItemList)
	se.data.Store(&empty)
	return se
}

// add inserts an ID if absent (idempotent).
func (s *stockEntry) add(id uint) {
	for {
		curPtr := s.data.Load()
		if curPtr == nil {
			// Initialize if somehow uninitialized (defensive, should not happen after constructor).
			empty := make(types.ItemList)
			if s.data.CompareAndSwap(nil, &empty) {
				curPtr = &empty
			} else {
				continue
			}
		}
		cur := *curPtr
		if _, exists := cur[id]; exists {
			return // already present; no mutation needed
		}

		// Create a new copy with additional id
		next := make(types.ItemList, len(cur)+1)
		for k := range cur {
			next[k] = struct{}{}
		}
		next[id] = struct{}{}

		if s.data.CompareAndSwap(curPtr, &next) {
			return
		}
		// CAS failed -> retry
	}
}

// remove deletes an ID if present (idempotent).
func (s *stockEntry) remove(id uint) {
	for {
		curPtr := s.data.Load()
		if curPtr == nil {
			return // nothing to remove
		}
		cur := *curPtr
		if _, exists := cur[id]; !exists {
			return // not present
		}

		if len(cur) == 1 {
			// Resulting set would be empty
			empty := make(types.ItemList)
			if s.data.CompareAndSwap(curPtr, &empty) {
				return
			}
			continue
		}

		next := make(types.ItemList, len(cur)-1)
		for k := range cur {
			if k != id {
				next[k] = struct{}{}
			}
		}

		if s.data.CompareAndSwap(curPtr, &next) {
			return
		}
		// CAS failed -> retry
	}
}

// snapshot returns a shallow copy of the current set.
// The returned map can be freely mutated by the caller.
func (s *stockEntry) snapshot() types.ItemList {
	curPtr := s.data.Load()
	if curPtr == nil {
		return make(types.ItemList)
	}
	cur := *curPtr
	cp := make(types.ItemList, len(cur))
	for id := range cur {
		cp[id] = struct{}{}
	}
	return cp
}
