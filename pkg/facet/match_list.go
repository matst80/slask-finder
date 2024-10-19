package facet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
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
			binary.Write(buf, binary.LittleEndian, math.Float64bits(f))
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
			(*d)[key] = str
			data = data[strLen:]
		case 1:
			i := int64(binary.BigEndian.Uint64(data[:8]))
			data = data[8:]
			(*d)[key] = int(i)
		case 2:
			f := binary.LittleEndian.Uint64(data[:8])
			data = data[8:]
			(*d)[key] = math.Float64frombits(f)
		}
	}
	*s = *d
	return nil
}
