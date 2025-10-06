package facet

import (
	"sort"

	"github.com/RoaringBitmap/roaring/v2"
)

/*
Legacy generic Bucket retained for IntegerField (int values).
Optimized ValueBucket implementation added for DecimalField refactor
storing monetary (float) values as integer cents for better performance,
reduced hashing, and faster range queries.
*/

// ---------------- Legacy (used by IntegerField) ----------------

type Bucket[V FieldNumberValue] struct {
	minValue V
	maxValue V
	values   map[V]*roaring.Bitmap
}

func (b *Bucket[V]) AddValueLink(value V, itemId uint32) {
	idList, ok := b.values[value]
	if b.minValue > value {
		b.minValue = value
	}
	if b.maxValue < value {
		b.maxValue = value
	}
	if !ok {
		idList = roaring.NewBitmap()
		idList.Add(itemId)
		b.values[value] = idList
	} else {
		idList.Add(itemId)
	}
}

func (b *Bucket[V]) RemoveValueLink(value V, id uint32) {
	idList, ok := b.values[value]
	if !ok {
		return
	}
	idList.Remove(id)
}

func MakeBucket[V FieldNumberValue](value V, itemId uint32) Bucket[V] {
	l := roaring.NewBitmap()
	l.Add(itemId)
	return Bucket[V]{
		values: map[V]*roaring.Bitmap{value: l},
	}
}

// ---------------- Optimized buckets for DecimalField ----------------

// valueEntry holds a single distinct value (in integer cents) + its item IDs
type valueEntry struct {
	value int64
	ids   *roaring.Bitmap
}

// ValueBucket groups multiple valueEntry items that fall into the same coarse bucket.
// entries is kept sorted on insert; merged is the OR of all ids in the bucket.
type ValueBucket struct {
	entries []valueEntry
	merged  *roaring.Bitmap
}

// NewValueBucket creates an empty bucket
func NewValueBucket() *ValueBucket {
	return &ValueBucket{
		entries: make([]valueEntry, 0, 8),
		merged:  roaring.NewBitmap(),
	}
}

// AddValue inserts (valueCents, itemId) into the bucket (valueCents must already be integer cents)
func (b *ValueBucket) AddValue(valueCents int64, itemId uint32) {
	// Binary search for value position
	i := sort.Search(len(b.entries), func(i int) bool {
		return b.entries[i].value >= valueCents
	})
	if i == len(b.entries) || b.entries[i].value != valueCents {
		// Insert new entry
		b.entries = append(b.entries, valueEntry{})
		copy(b.entries[i+1:], b.entries[i:])
		b.entries[i] = valueEntry{
			value: valueCents,
			ids:   roaring.NewBitmap(),
		}
	}
	b.entries[i].ids.Add(itemId)
	b.merged.Add(itemId)
}

// RemoveValue removes (valueCents, itemId) from the bucket (lazy merged maintenance)
func (b *ValueBucket) RemoveValue(valueCents int64, itemId uint32) {
	i := sort.Search(len(b.entries), func(i int) bool {
		return b.entries[i].value >= valueCents
	})
	if i == len(b.entries) || b.entries[i].value != valueCents {
		return
	}
	b.entries[i].ids.Remove(itemId)
	// NOTE: merged is not decremented (lazy). If high deletion accuracy is needed,
	// rebuild merged when many removals accumulate.
}

// RangeUnion unions all ids whose value lies in [minCents, maxCents] (inclusive) into acc.
func (b *ValueBucket) RangeUnion(minCents, maxCents int64, acc *roaring.Bitmap) {
	if len(b.entries) == 0 {
		return
	}
	start := sort.Search(len(b.entries), func(i int) bool {
		return b.entries[i].value >= minCents
	})
	for j := start; j < len(b.entries); j++ {
		ve := b.entries[j]
		if ve.value > maxCents {
			break
		}
		acc.Or(ve.ids)
	}
}

// ---------------- Bucket addressing & resolution ----------------

// Bits_To_Shift controls coarse bucket resolution.
// With integer cents, 1<<9 (512 cents =~ $5.12) per bucket.
const Bits_To_Shift = 9

// GetBucket returns the bucket id for an integer-like value
func GetBucket[V float64 | int | float32](value V) int {
	return int(value) >> Bits_To_Shift
}

// GetBucketFromCents returns bucket for integer cents representation
func GetBucketFromCents(valueCents int64) int {
	return int(valueCents) >> Bits_To_Shift
}
