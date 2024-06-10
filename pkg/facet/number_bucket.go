package facet

type NumberBucket struct {
	IdList
}

const Bits_To_Shift = 8 // 256 per bucket

func GetBucket[V float64 | int64 | int | float32](value V) int64 {
	r := int64(value) >> Bits_To_Shift
	return r
}
