package types

import (
	"sort"

	"github.com/RoaringBitmap/roaring/v2"
)

/*
ItemList

A compatibility wrapper around a *roaring.Bitmap providing (mostly) the
same semantics as the legacy map[uint]struct{}-based ItemList while
leveraging roaring bitmaps for:

  - Dramatically lower memory footprint for large sparse ID sets
  - Fast intersections / unions / differences
  - Efficient cardinality queries

Design choices:
  - Zero value is usable: a zero ItemList lazily allocates its internal bitmap.
  - All mutating operations ensure() the underlying bitmap.
  - Methods use pointer receivers; calling them on a value is still valid
    (Go will take the address of an addressable value automatically).
  - Nil pointer receiver is tolerated (treated as empty / no-op).

Semantics:
  - Merge(other)        : set union            (A = A ∪ B)
  - Intersect(other)    : in-place intersection (A = A ∩ B)
  - Exclude(other)      : relative complement   (A = A \ B)
  - AddId(id)           : add single element
  - HasIntersection(b)  : true if (A ∩ B) ≠ ∅
  - IntersectionLen(b)  : |A ∩ B|
  - Len()               : |A|
  - Contains(id)        : membership test
  - ForEach(fn)         : ordered iteration (ascending IDs)
  - ToSlice()           : ordered slice of IDs
  - ToMap()             : map[uint]struct{} (only when needed for legacy code)
  - AddAllFrom(other)   : union (same as Merge)
  - MergeMap(m)         : merge keys from map[uint]struct{}
  - FromBitmap / OrBitmap helpers for integration with roaring-native paths.

NOTE:
  - Operations that produce temporary clones (HasIntersection / IntersectionLen)
    only allocate when both sets are non-empty; they pick the smaller set
    to reduce memory churn.
*/

type ItemList struct {
	bm *roaring.Bitmap
}

// ensure lazily allocates the bitmap.
func (l *ItemList) ensure() {
	if l != nil && l.bm == nil {
		l.bm = roaring.NewBitmap()
	}
}

// NewItemList constructs an empty list.
func NewItemList() *ItemList {
	return &ItemList{bm: roaring.NewBitmap()}
}

// Clone returns a deep copy.
func (l *ItemList) Clone() *ItemList {
	if l == nil || l.bm == nil {
		return NewItemList()
	}
	return &ItemList{bm: l.bm.Clone()}
}

// FromBitmap wraps (clones) an existing roaring bitmap.
func FromBitmap(bm *roaring.Bitmap) *ItemList {
	if bm == nil {
		return NewItemList()
	}
	return &ItemList{bm: bm.Clone()}
}

// Bitmap exposes the internal bitmap (read-only!). Never mutate it directly.
func (l *ItemList) Bitmap() *roaring.Bitmap {
	if l == nil {
		return nil
	}
	return l.bm
}

// AddId adds a single id.
func (l *ItemList) AddId(id uint) {
	if l == nil {
		return
	}
	l.ensure()
	l.bm.Add(uint32(id))
}

// Merge (union) with other (A = A ∪ B).
func (l *ItemList) Merge(other *ItemList) {
	if l == nil || other == nil || other.bm == nil || other.bm.IsEmpty() {
		return
	}
	l.ensure()
	l.bm.Or(other.bm)
}

// AddAllFrom is an alias to Merge for compatibility.
func (l *ItemList) AddAllFrom(other *ItemList) {
	l.Merge(other)
}

// Intersect in-place (A = A ∩ B).
func (l *ItemList) Intersect(other *ItemList) {
	if l == nil || l.bm == nil || other == nil || other.bm == nil {
		return
	}
	l.bm.And(other.bm)
}

// Exclude subtracts other (A = A \ B).
func (l *ItemList) Exclude(other *ItemList) {
	if l == nil || l.bm == nil || other == nil || other.bm == nil || other.bm.IsEmpty() {
		return
	}
	l.bm.AndNot(other.bm)
}

// HasIntersection returns true if any element overlaps.
func (l *ItemList) HasIntersection(other *ItemList) bool {
	if l == nil || other == nil || l.bm == nil || other.bm == nil {
		return false
	}
	if l.bm.IsEmpty() || other.bm.IsEmpty() {
		return false
	}
	// Optimize by cloning smaller
	if l.bm.GetCardinality() > other.bm.GetCardinality() {
		l, other = other, l
	}
	tmp := l.bm.Clone()
	tmp.And(other.bm)
	return !tmp.IsEmpty()
}

// IntersectionLen returns |A ∩ B|.
func (l *ItemList) IntersectionLen(other *ItemList) int {
	if l == nil || other == nil || l.bm == nil || other.bm == nil || l.bm.IsEmpty() || other.bm.IsEmpty() {
		return 0
	}
	if l.bm.GetCardinality() > other.bm.GetCardinality() {
		l, other = other, l
	}
	tmp := l.bm.Clone()
	tmp.And(other.bm)
	return int(tmp.GetCardinality())
}

// Len returns cardinality as int.
func (l *ItemList) Len() int {
	if l == nil || l.bm == nil {
		return 0
	}
	return int(l.bm.GetCardinality())
}

// Cardinality returns cardinality as uint64.
func (l *ItemList) Cardinality() uint64 {
	if l == nil || l.bm == nil {
		return 0
	}
	return l.bm.GetCardinality()
}

// Contains tests membership.
func (l *ItemList) Contains(id uint) bool {
	if l == nil || l.bm == nil {
		return false
	}
	return l.bm.Contains(uint32(id))
}

// ForEach iterates in ascending order; stop early if fn returns false.
func (l *ItemList) ForEach(fn func(id uint) bool) {
	if l == nil || l.bm == nil || fn == nil {
		return
	}
	it := l.bm.Iterator()
	for it.HasNext() {
		if !fn(uint(it.Next())) {
			return
		}
	}
}

// ToSlice returns all ids (ascending).
func (l *ItemList) ToSlice() []uint {
	if l == nil || l.bm == nil {
		return []uint{}
	}
	out32 := l.bm.ToArray() // already ascending
	out := make([]uint, len(out32))
	for i, v := range out32 {
		out[i] = uint(v)
	}
	return out
}

// ToMap converts to map[uint]struct{} (only use when legacy APIs demand it).
func (l *ItemList) ToMap() map[uint]struct{} {
	m := make(map[uint]struct{}, l.Len())
	l.ForEach(func(id uint) bool {
		m[id] = struct{}{}
		return true
	})
	return m
}

// MergeMap merges keys from a map[uint]struct{}.
func (l *ItemList) MergeMap(m map[uint]struct{}) {
	if l == nil || len(m) == 0 {
		return
	}
	l.ensure()
	// Copy keys into slice for potential batch addition improvements (keeps sorted insert stable)
	keys := make([]uint32, 0, len(m))
	for k := range m {
		keys = append(keys, uint32(k))
	}
	// roaring.AddMany does not guarantee deduped order; sort for compression effectiveness.
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	l.bm.AddMany(keys)
}

// OrBitmap unions an external roaring bitmap.
func (l *ItemList) OrBitmap(bm *roaring.Bitmap) {
	if bm == nil || bm.IsEmpty() || l == nil {
		return
	}
	l.ensure()
	l.bm.Or(bm)
}

// FromSlice builds an ItemList from ids (deduplicates & sorts).
func FromSlice(ids []uint) *ItemList {
	il := NewItemList()
	if len(ids) == 0 {
		return il
	}
	tmp := make([]uint32, 0, len(ids))
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		tmp = append(tmp, uint32(id))
	}
	sort.Slice(tmp, func(i, j int) bool { return tmp[i] < tmp[j] })
	il.bm.AddMany(tmp)
	return il
}

// Equals tests set equality (helper for tests).
func (l *ItemList) Equals(other *ItemList) bool {
	if l == other {
		return true
	}
	if l == nil || other == nil {
		return l.Len() == other.Len()
	}
	if l.Len() != other.Len() {
		return false
	}
	// Compare by cloning smaller & intersecting; if cardinality unchanged sets equal
	if l.bm.GetCardinality() > other.bm.GetCardinality() {
		l, other = other, l
	}
	tmp := l.bm.Clone()
	tmp.And(other.bm)
	return tmp.GetCardinality() == l.bm.GetCardinality()
}

// Reset clears the bitmap.
func (l *ItemList) Reset() {
	if l == nil {
		return
	}
	if l.bm == nil {
		return
	}
	l.bm.Clear()
}

// IsEmpty reports if the set is empty.
func (l *ItemList) IsEmpty() bool {
	return l == nil || l.bm == nil || l.bm.IsEmpty()
}
