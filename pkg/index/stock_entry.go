package index

import (
	"sync/atomic"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/matst80/slask-finder/pkg/types"
)

// stockEntry is a lock-free, copy-on-write container for a set of item IDs
// associated with a single stock location. It uses an atomic pointer to an
// immutable roaring.Bitmap. Writers clone, mutate, and CAS; readers access
// the bitmap directly with no locking. This is optimized for high read /
// low write workloads.
type stockEntry struct {
	ptr atomic.Pointer[roaring.Bitmap]
}

// newStockEntry returns a new, empty stockEntry.
func newStockEntry() *stockEntry {
	bm := roaring.NewBitmap()
	se := &stockEntry{}
	se.ptr.Store(bm)
	return se
}

// load returns the current immutable bitmap pointer.
func (s *stockEntry) load() *roaring.Bitmap {
	return s.ptr.Load()
}

// add inserts id if not already present (idempotent).
func (s *stockEntry) add(id uint) {
	uid := uint32(id)
	for {
		cur := s.load()
		if cur.Contains(uid) {
			return
		}
		next := cur.Clone()
		before := next.GetCardinality()
		next.Add(uid)
		if next.GetCardinality() == before {
			return // no change; race where another writer inserted already
		}
		if s.ptr.CompareAndSwap(cur, next) {
			return
		}
		// CAS failed; retry
	}
}

// remove deletes id if present (idempotent).
func (s *stockEntry) remove(id uint) {
	uid := uint32(id)
	for {
		cur := s.load()
		if !cur.Contains(uid) {
			return
		}
		next := cur.Clone()
		next.Remove(uid)
		if s.ptr.CompareAndSwap(cur, next) {
			return
		}
		// CAS failed; retry
	}
}

// snapshot builds and returns a types.ItemList containing all current ids.
// Adapted for roaring-backed ItemList (no direct map allocation).
func (s *stockEntry) snapshot() types.ItemList {
	cur := s.load()
	var result types.ItemList
	it := cur.Iterator()
	for it.HasNext() {
		result.AddId(uint(it.Next()))
	}
	return result
}

// bitmap returns the underlying immutable roaring bitmap. Callers MUST NOT
// mutate it. (Provided for advanced intersection / union operations.)
func (s *stockEntry) bitmap() *roaring.Bitmap {
	return s.load()
}

// contains returns true if id is in the set.
func (s *stockEntry) contains(id uint) bool {
	return s.load().Contains(uint32(id))
}

// cardinality returns number of items in the set.
func (s *stockEntry) cardinality() uint64 {
	return s.load().GetCardinality()
}

// toItemListUnsafe converts a roaring bitmap directly to an ItemList without cloning
// the bitmap first; only used internally if needed for optimization. Not currently used.
// func (s *stockEntry) toItemListUnsafe() types.ItemList {
// 	cur := s.load()
// 	result := make(types.ItemList, cur.GetCardinality())
// 	it := cur.Iterator()
// 	for it.HasNext() {
// 		result[uint(it.Next())] = struct{}{}
// 	}
// 	return result
// }

// Notes on memory usage:
// - roaring.Bitmap stores IDs in 16-bit keyed containers (array or bitmap)
//   which adapt to sparsity/density and are typically far more memory efficient
//   than a generic map[uint]struct{} for large sets.
// - Each write (add/remove) clones only the affected containers (Clone clones
//   the structure, but containers are copy-on-write internally in roaring).
// - For extremely write-heavy scenarios a simple mutex-protected mutable
//   structure could be faster; in high read / low write this design excels.
//
// Potential future optimization:
// Provide direct intersection utilities operating on *roaring.Bitmap to
// avoid converting to ItemList where not necessary.
