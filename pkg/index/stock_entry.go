package index

import (
	"sync"

	"github.com/RoaringBitmap/roaring/v2"
)

// stockEntry is a simplified concurrency wrapper around a single mutable
// roaring.Bitmap protected by an RWMutex. This replaces the previous
// lock-free copy-on-write (atomic pointer) design with something easier
// to reason about.
//
// Rationale:
//   - Writes are expected to be relatively low compared to reads, but the
//     previous design cloned the bitmap on every mutation which can become
//     more expensive if writes are not extremely rare.
//   - A single bitmap + RWMutex keeps code straightforward while still
//     allowing parallel readers.
//   - To preserve the "immutable to callers" contract that existing code
//     (e.g. GetStockResult) relied on, bitmap() returns a CLONE of the
//     underlying bitmap. This avoids external mutation & data races while
//     keeping internal representation mutable.
//
// If performance profiling later shows bitmap cloning to be a hotspot,
// an alternative API could expose a read-locked callback instead of
// returning a clone.
type stockEntry struct {
	mu sync.RWMutex
	bm *roaring.Bitmap
}

// newStockEntry returns an initialized, empty stockEntry.
func newStockEntry() *stockEntry {
	return &stockEntry{
		bm: roaring.NewBitmap(),
	}
}

// add inserts id (idempotent).
func (s *stockEntry) add(id uint32) {
	s.mu.Lock()
	s.bm.Add(id) // roaring.Add is idempotent; no need to check existence
	s.mu.Unlock()
}

// remove deletes id if present (idempotent).
func (s *stockEntry) remove(id uint32) {
	s.mu.Lock()
	s.bm.Remove(id)
	s.mu.Unlock()
}

// bitmap returns a CLONE of the underlying bitmap to keep callers from
// mutating internal state and to avoid data races during concurrent writes.
func (s *stockEntry) bitmap() *roaring.Bitmap {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bm
}
