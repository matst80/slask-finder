package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type MapStock struct {
	data map[string]uint16
}

type Stock interface {
	GetStock(string) map[string]uint16
}

func NewMapStock() *MapStock {
	return &MapStock{
		data: make(map[string]uint16),
	}
}

func (f *MapStock) GetStock() map[string]uint16 {
	return f.data
}

func (f *MapStock) SetStock(id string, value uint16) error {
	if id == "" {
		return errors.New("id cannot be empty")
	}
	if value == 0 {
		delete(f.data, id)
		return nil
	}
	f.data[id] = value
	return nil
}

// MarshalJSON implements a low-allocation JSON object serializer.
// Output format: {"<id>":<value>, ...}
func (f MapStock) MarshalJSON() ([]byte, error) {
	// Pre-size buffer roughly (heuristic).
	var buf bytes.Buffer
	buf.Grow((len(f.data))*24 + 2)
	buf.WriteByte('{')
	first := true

	writeComma := func() {
		if first {
			first = false
		} else {
			buf.WriteByte(',')
		}
	}
	// Strings
	for id, value := range f.data {
		if value == 0 {
			continue
		}
		writeComma()
		buf.WriteByte('"')
		buf.WriteString(id)
		buf.WriteString(`":`)
		buf.WriteString(fmt.Sprintf(`"%d"`, value))
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON parses the custom facet object.
// Accepts values: string, number, or array of strings (joined with ", ").
func (f *MapStock) UnmarshalJSON(data []byte) error {
	// Reset slices (allow reuse of underlying arrays).
	f.data = map[string]uint16{}
	var id64 uint64
	i := 0
	skipWS := func() {
		for i < len(data) {
			switch data[i] {
			case ' ', '\n', '\r', '\t':
				i++
			default:
				return
			}
		}
	}
	parseString := func() (string, error) {
		if i >= len(data) || data[i] != '"' {
			return "", errors.New("expected '\"' at string start")
		}
		i++
		start := i

		for i < len(data) {
			c := data[i]

			if c == '"' {
				s := data[start:i]
				i++
				return string(s), nil
			}
			i++
		}
		return "", errors.New("unterminated string")
	}

	skipWS()
	if i >= len(data) || data[i] != '{' {
		return errors.New("expected '{'")
	}
	i++
	skipWS()
	if i < len(data) && data[i] == '}' {
		i++
		return nil
	}

	for {
		skipWS()
		keyStr, err := parseString()
		if err != nil {
			return err
		}
		// id64, err := strconv.ParseUint(keyStr, 10, 64)
		// if err != nil {
		// 	return errors.New("facet key is not an unsigned integer: " + keyStr)
		// }
		skipWS()
		if i >= len(data) || data[i] != ':' {
			return errors.New("expected ':' after key")
		}
		i++
		skipWS()
		if i >= len(data) {
			return errors.New("unexpected end after ':'")
		}
		switch data[i] {
		case '"':
			s, err := parseString()
			if err != nil {
				return err
			}

			switch s {
			case "100+":
				id64 = 100
			case "10+":
				id64 = 10
			case "25+":
				id64 = 25
			case "5+":
				id64 = 5
			case "1+":
				id64 = 1
			case "<10":
				id64 = 5
			case "\u003c5":
				id64 = 5
			default:
				id64, err = strconv.ParseUint(s, 10, 64)
				if err != nil {
					return errors.New("facet value is not an unsigned integer: " + s)
				}
			}
			if id64 > 0 {
				f.data[keyStr] = uint16(id64)
			}

		}
		skipWS()
		if i >= len(data) {
			return errors.New("unexpected end after value")
		}
		if data[i] == ',' {
			i++
			continue
		}
		if data[i] == '}' {
			i++
			break
		}
		return errors.New("expected ',' or '}' after value")
	}
	return nil
}

// Ensure ItemFields implements json.Marshaler / json.Unmarshaler.
var (
	_ json.Marshaler   = (*MapStock)(nil)
	_ json.Unmarshaler = (*MapStock)(nil)
)
