package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// ItemFields stores string and numeric facets separately to avoid interface{} boxing.
// It implements custom JSON marshal / unmarshal optimized for low allocations.
type ItemFields struct {
	keyFacets    map[FacetId]string
	numberFacets map[FacetId]float64
}

func NewItemFields() *ItemFields {
	return &ItemFields{
		keyFacets:    make(map[FacetId]string),
		numberFacets: make(map[FacetId]float64),
	}
}

func (f ItemFields) GetNumberFieldValue(id FacetId) (float64, bool) {
	v, ok := f.numberFacets[id]
	if ok {
		return v, true
	}
	return 0, false
}

func (f ItemFields) GetStringsFieldValue(id FacetId) ([]string, bool) {

	v, ok := f.keyFacets[id]
	if ok {
		return strings.Split(v, ";"), true
	}
	return nil, false
}

func (f ItemFields) GetStringFieldValue(id FacetId) (string, bool) {
	v, ok := f.keyFacets[id]
	if ok {
		return v, true
	}
	return "", false
}

func (f ItemFields) GetNumberFields() map[FacetId]float64 {
	return f.numberFacets
}

func (f ItemFields) GetStringFields() map[FacetId]string {
	return f.keyFacets
}

// // GetFacets materializes all facets into a map[uint]any. This allocates;
// // prefer using GetFacetValue when possible.
// func (f *ItemFields) GetFacets() map[uint]any {
// 	m := make(map[uint]any, len(f.keyFacets)+len(f.numberFacets))
// 	for _, kv := range f.keyFacets {
// 		m[kv.ID] = kv.Value
// 	}
// 	for _, kv := range f.numberFacets {
// 		m[kv.ID] = kv.Value
// 	}
// 	return m
// }

// upsertString inserts / updates a string facet.
func (f *ItemFields) upsertString(id uint, val string) {
	f.keyFacets[FacetId(id)] = val
}

// upsertNumber inserts / updates a numeric facet.
func (f *ItemFields) upsertNumber(id uint, val float64) {
	f.numberFacets[FacetId(id)] = val

}

// MarshalJSON implements a low-allocation JSON object serializer.
// Output format: {"<id>":<value>, ...}
func (f ItemFields) MarshalJSON() ([]byte, error) {
	// Pre-size buffer roughly (heuristic).
	var buf bytes.Buffer
	buf.Grow((len(f.keyFacets)+len(f.numberFacets))*24 + 2)
	buf.WriteByte('{')
	first := true
	var tmp []byte
	writeComma := func() {
		if first {
			first = false
		} else {
			buf.WriteByte(',')
		}
	}
	// Strings
	for id, value := range f.keyFacets {
		writeComma()
		buf.WriteByte('"')
		buf.WriteString(strconv.FormatUint(uint64(id), 10))
		buf.WriteString(`":`)
		if strings.Contains(value, ";") {
			parts := strings.Split(value, ";")
			buf.WriteByte('[')
			for i, part := range parts {
				if i > 0 {
					buf.WriteByte(',')
				}
				tmp = strconv.AppendQuote(tmp[:0], part)
				buf.Write(tmp)
			}
			buf.WriteByte(']')
		} else {
			tmp = strconv.AppendQuote(tmp[:0], value)
			buf.Write(tmp)
		}
	}
	// Numbers
	for id, value := range f.numberFacets {
		writeComma()
		buf.WriteByte('"')
		buf.WriteString(strconv.FormatUint(uint64(id), 10))
		buf.WriteString(`":`)
		tmp = strconv.AppendFloat(tmp[:0], value, 'f', -1, 64)
		buf.Write(tmp)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON parses the custom facet object.
// Accepts values: string, number, or array of strings (joined with ", ").
func (f *ItemFields) UnmarshalJSON(data []byte) error {
	// Reset slices (allow reuse of underlying arrays).
	f.keyFacets = map[FacetId]string{}
	f.numberFacets = map[FacetId]float64{}

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
		hasEscape := false
		for i < len(data) {
			c := data[i]
			if c == '\\' {
				hasEscape = true
				i += 2
				continue
			}
			if c == '"' {
				s := data[start:i]
				i++
				if hasEscape {
					// Fallback to standard unescape
					var out string
					if err := json.Unmarshal(data[start-1:i], &out); err != nil {
						return "", err
					}
					return out, nil
				}
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
		id64, err := strconv.ParseUint(keyStr, 10, 64)
		if err != nil {
			return errors.New("facet key is not an unsigned integer: " + keyStr)
		}
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
			f.upsertString(uint(id64), s)
		case '[':
			// array of strings -> joined
			i++
			skipWS()
			if i >= len(data) {
				return errors.New("unexpected end in array")
			}
			var sb strings.Builder
			firstElem := true
			for {
				skipWS()
				if i < len(data) && data[i] == ']' {
					i++
					break
				}
				elem, err := parseString()
				if err != nil {
					return err
				}
				if !firstElem {
					sb.WriteString(";")
				} else {
					firstElem = false
				}
				sb.WriteString(elem)
				skipWS()
				if i >= len(data) {
					return errors.New("unexpected end in array")
				}
				if data[i] == ',' {
					i++
					continue
				}
				if data[i] == ']' {
					i++
					break
				}
				return errors.New("expected ',' or ']' in array")
			}
			f.upsertString(uint(id64), sb.String())
		default:
			// Parse number
			startNum := i
			if data[i] == '-' {
				i++
			}
			for i < len(data) && (data[i] >= '0' && data[i] <= '9') {
				i++
			}
			if i < len(data) && data[i] == '.' {
				i++
				for i < len(data) && (data[i] >= '0' && data[i] <= '9') {
					i++
				}
			}
			numBytes := data[startNum:i]
			if len(numBytes) == 0 {
				return errors.New("invalid number")
			}
			num, err := strconv.ParseFloat(string(numBytes), 64)
			if err != nil {
				return err
			}
			f.upsertNumber(uint(id64), num)
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
	_ json.Marshaler   = (*ItemFields)(nil)
	_ json.Unmarshaler = (*ItemFields)(nil)
)
