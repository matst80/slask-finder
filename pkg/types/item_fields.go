package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type ItemFacets interface {
	SetValue(id uint, value interface{})
	GetFacetValue(facetId uint) (interface{}, bool)
	GetFacets() map[uint]interface{}
}

type ItemFields map[uint]interface{}

func (b ItemFields) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	l := len(b)

	MustWrite(buf, uint64(l))

	for k, v := range b {
		MustWrite(buf, uint64(k))
		switch typed := v.(type) {
		case string:
			MustWrite(buf, uint8(0))
			l := uint8(len(typed))
			MustWrite(buf, l)
			MustWrite(buf, []byte(typed[:l]))

		case int:

			MustWrite(buf, uint8(1))
			MustWrite(buf, int64(typed))

		case float64:

			MustWrite(buf, uint8(2))
			binary.Write(buf, binary.LittleEndian, math.Float64bits(typed))

		case []string:
			MustWrite(buf, uint8(3))
			l := uint8(len(typed))
			MustWrite(buf, l)
			for _, str := range typed {
				strLen := uint8(len(str))
				MustWrite(buf, strLen)
				MustWrite(buf, []byte(str[:strLen]))
			}
		}
	}

	return buf.Bytes(), nil
}

func (b ItemFields) GetFacetValue(id uint) (interface{}, bool) {
	v, ok := b[id]
	return v, ok
}

func (b ItemFields) GetFacets() map[uint]interface{} {
	return b
}

func (b ItemFields) SetValue(id uint, value interface{}) {
	b[id] = value
}

func MustWrite(w io.Writer, data interface{}) {
	if err := binary.Write(w, binary.BigEndian, data); err != nil {
		panic(fmt.Errorf("failed to write binary data: %v", data).Error())
	}
}

func (s *ItemFields) UnmarshalBinary(data []byte) error {
	b := bytes.NewBuffer(data)
	var len uint64
	err := binary.Read(b, binary.BigEndian, &len)
	if err != nil {
		return err
	}
	d := ItemFields{}
	//data = data[8:]
	for i := 0; i < int(len); i++ {
		var key uint64
		var typ uint8
		binary.Read(b, binary.BigEndian, &key)
		binary.Read(b, binary.BigEndian, &typ)

		switch typ {
		case 0:
			var strLen uint8
			binary.Read(b, binary.BigEndian, &strLen)
			stringBytes := make([]byte, strLen)
			binary.Read(b, binary.BigEndian, &stringBytes)

			d[uint(key)] = string(stringBytes)

		case 1:
			var i int64
			binary.Read(b, binary.BigEndian, &i)
			//i := int64(binary.BigEndian.Uint64(data[:8]))
			//data = data[8:]
			d[uint(key)] = int(i)
		case 2:
			var fbits uint64
			binary.Read(b, binary.LittleEndian, &fbits)
			d[uint(key)] = math.Float64frombits(fbits)

		case 3:
			var arrayLength uint8
			binary.Read(b, binary.BigEndian, &arrayLength)
			if arrayLength == 0 {
				d[uint(key)] = []string{}
				continue
			}
			values := make([]string, arrayLength)
			for j := range int(arrayLength) {
				var strLen uint8
				binary.Read(b, binary.BigEndian, &strLen)
				stringBytes := make([]byte, strLen)
				binary.Read(b, binary.BigEndian, &stringBytes)
				values[j] = string(stringBytes)
			}
			d[uint(key)] = values
		}
	}
	*s = d
	return nil
}
