package facet

import (
	"github.com/RoaringBitmap/roaring/v2"
)

/*
number_helpers.go

This file centralizes helpers that were previously duplicated between
integer_field.go and number_field.go (decimal). The common patterns are:

1. Bucket lower / upper bound helpers (previously:
   - DecimalField: bucketLowerBoundCents / bucketUpperBoundCents
   - IntegerField: bucketLowerBoundInt   / bucketUpperBoundInt
   They are identical except for naming. We expose generic versions now.

2. Scanning buckets to find the minimum / maximum matching value for a
   provided filter bitmap. Both IntegerField.GetExtents and
   DecimalField.GetExtents performed almost identical logic; the only
   difference is the value domain (int vs integer cents). Since both
   store values in ValueBucket as int64 (DecimalField already in cents;
   IntegerField inserts int values cast to int64), we can unify the scan.

Usage (existing code can be refactored to):
   minV, okMin := ScanMinValue(buckets, startBucket, endBucket, filterBM)
   maxV, okMax := ScanMaxValue(buckets, startBucket, endBucket, filterBM)

The caller remains responsible for:
   - Deriving startBucket / endBucket (with GetBucket / GetBucketFromCents)
   - Converting returned int64 to the appropriate external type
     (e.g. cents -> float, or int64 -> int)

No behavioral changes are introduced; this is purely a consolidation.
*/

// BucketLowerBound returns the inclusive lower bound for a coarse bucket
// in the integer domain used by both integer and decimal (cents) fields.
func BucketLowerBound(bucket int) int64 {
	return int64(bucket << Bits_To_Shift)
}

// BucketUpperBound returns the inclusive upper bound for a coarse bucket
// (i.e. start of next bucket minus one).
func BucketUpperBound(bucket int) int64 {
	return int64(((bucket + 1) << Bits_To_Shift) - 1)
}

// ScanMinValue performs an ascending bucket scan (startBucket -> endBucket)
// returning the first (smallest) value whose posting list intersects filterBM.
// Values are stored sorted inside each ValueBucket, so the first match inside
// a qualifying bucket is the global minimum still to be found.
func ScanMinValue(
	buckets map[int]*ValueBucket,
	startBucket int,
	endBucket int,
	filterBM *roaring.Bitmap,
) (int64, bool) {

	for bId := startBucket; bId <= endBucket; bId++ {
		b, ok := buckets[bId]
		if !ok || b == nil || b.merged == nil || b.merged.IsEmpty() {
			continue
		}
		if !b.merged.Intersects(filterBM) {
			continue
		}
		for _, ve := range b.entries { // ascending
			if ve.ids.Intersects(filterBM) {
				return ve.value, true
			}
		}
	}
	return 0, false
}

// ScanMaxValue performs a descending bucket scan (endBucket -> startBucket)
// returning the first (largest) value whose posting list intersects filterBM.
// Because entries are stored sorted ascending inside a bucket, we iterate
// each bucket's entries in reverse to obtain the largest match quickly.
func ScanMaxValue(
	buckets map[int]*ValueBucket,
	startBucket int,
	endBucket int,
	filterBM *roaring.Bitmap,
) (int64, bool) {

	for bId := endBucket; bId >= startBucket; bId-- {
		b, ok := buckets[bId]
		if !ok || b == nil || b.merged == nil || b.merged.IsEmpty() {
			continue
		}
		if !b.merged.Intersects(filterBM) {
			continue
		}
		for i := len(b.entries) - 1; i >= 0; i-- {
			ve := b.entries[i]
			if ve.ids.Intersects(filterBM) {
				return ve.value, true
			}
		}
		if bId == startBucket {
			break
		}
	}
	return 0, false
}

// ScanMinMax is a convenience helper that attempts to find both ends
// using separate directional scans. For typical data (where the min and
// max are far apart) this is comparable to performing two scans directly.
// Provided mainly for completeness; callers that already need early
// access to min before deciding to search for max may prefer separate calls.
func ScanMinMax(
	buckets map[int]*ValueBucket,
	startBucket int,
	endBucket int,
	filterBM *roaring.Bitmap,
) (minValue int64, maxValue int64, ok bool) {

	minV, okMin := ScanMinValue(buckets, startBucket, endBucket, filterBM)
	if !okMin {
		return 0, 0, false
	}
	maxV, okMax := ScanMaxValue(buckets, startBucket, endBucket, filterBM)
	if !okMax {
		// This should not happen logically if min found, but keep defensive.
		return 0, 0, false
	}
	return minV, maxV, true
}
