package facet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

type ItemFields map[uint]interface{}

func (b ItemFields) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	l := len(b)

	MustWrite(buf, uint64(l))

	for k, v := range b {
		MustWrite(buf, uint64(k))
		str, ok := v.(string)
		if ok {
			MustWrite(buf, uint8(0))
			l := uint8(len(str))
			MustWrite(buf, l)
			MustWrite(buf, []byte(str[:l]))
			continue
		}
		i, ok := v.(int)
		if ok {
			MustWrite(buf, uint8(1))
			MustWrite(buf, int64(i))
			continue
		}
		f, ok := v.(float64)
		if ok {
			MustWrite(buf, uint8(2))
			MustWrite(buf, f)
			continue
		}

	}

	return buf.Bytes(), nil
}

func MustWrite(w io.Writer, data interface{}) {
	if err := binary.Write(w, binary.BigEndian, data); err != nil {
		panic(fmt.Errorf("failed to write binary data: %v", data).Error())
	}
}

// implement encoding.BinaryUnmarshaler interface for DemoStruct
func (s *ItemFields) UnmarshalBinary(data []byte) error {
	d := &ItemFields{}
	len := binary.BigEndian.Uint64(data[:8])
	data = data[8:]
	for i := 0; i < int(len); i++ {
		key := uint(binary.BigEndian.Uint64(data[:8]))
		data = data[8:]
		typ := data[0]
		data = data[1:]
		switch typ {
		case 0:
			strLen := data[0]
			data = data[1:]
			str := string(data[:strLen])
			(*d)[key] = strings.Trim(str, "\x00")
			data = data[strLen:]
		case 1:
			i := int64(binary.BigEndian.Uint64(data[:8]))
			data = data[8:]
			(*d)[key] = int(i)
		case 2:
			f := binary.BigEndian.Uint64(data[:8])
			data = data[8:]
			(*d)[key] = float64(f)
		}
	}
	*s = *d
	return nil
}

//type MatchList map[uint]*ItemFields[FieldValue]

// func (r *MatchList) SortedIds(srt *SortIndex, maxItems int) []uint {
// 	return srt.SortMatch(*r, maxItems)
// }

// func (a MatchList) Intersect(b MatchList) {
// 	for id := range a {
// 		_, ok := b[id]
// 		if !ok {
// 			delete(a, id)
// 		}
// 	}
// }

// func (i MatchList) Merge(other *MatchList) {
// 	maps.Copy(i, *other)
// }

// func MakeIntersectResult(r chan MatchList, len int) *MatchList {

// 	if len == 0 {
// 		return &MatchList{}
// 	}
// 	first := <-r
// 	for i := 1; i < len; i++ {
// 		first.Intersect(<-r)
// 	}
// 	close(r)
// 	return &first
// }
