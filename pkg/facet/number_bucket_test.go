package facet

import (
	"testing"
)

func TestGetBucket(t *testing.T) {

	intValues := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, v := range intValues {
		if GetBucket(v) != 0 {
			t.Errorf("Expected 0")
		}
	}

	largeIntValues := []int{1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010}

	for _, v := range largeIntValues {
		bucket := GetBucket(v)
		if bucket >= v {
			t.Errorf("Expected 10, got %v from %v", bucket, v)
		}
	}

	floatValues := []float64{1.1, 2.2, 3.3, 4.4, 5.5, 6.6, 7.7, 8.8, 9.9, 10.10}
	for _, v := range floatValues {
		if GetBucket(v) != 0 {
			t.Errorf("Expected 0")
		}
	}

	largeFloatValues := []float64{10000, 10005, 10010, 10015, 10000, 10002}
	expected := int(10000 >> Bits_To_Shift)
	for _, v := range largeFloatValues {
		bucket := GetBucket(v)
		if bucket != expected {
			t.Errorf("Expected %v, got %v from %v", expected, bucket, v)
		}
	}
}
