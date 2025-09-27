//go:build jsonv2

package types

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"sort"
	"strconv"
)

// Flags to tweak behavior at runtime in benchmarks/tests.
// Set from tests or init in other packages.
var EmitCompactItemFields = false  // if true: use [[id,value],...] form
var DeterministicItemFields = true // if true: sort IDs ascending before encoding

// MarshalJSONTo provides a faster custom path for json/v2.
// It supports either legacy object form or a compact array-of-pairs form.
func (f ItemFields) MarshalJSONTo(enc *jsontext.Encoder) error {
	if f == nil {
		return enc.WriteToken(jsontext.Null)
	}

	if !EmitCompactItemFields { // legacy object form {"4":123, ...}
		ids := make([]uint, 0, len(f))
		for id := range f {
			ids = append(ids, id)
		}
		if DeterministicItemFields {
			sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		}
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		for _, id := range ids {
			if err := enc.WriteToken(jsontext.String(strconv.FormatUint(uint64(id), 10))); err != nil {
				return err
			}
			if err := writeArbitraryValue(enc, f[id]); err != nil {
				return err
			}
		}
		return enc.WriteToken(jsontext.EndObject)
	}

	// compact form: [[id,value], ...]
	ids := make([]uint, 0, len(f))
	for id := range f {
		ids = append(ids, id)
	}
	if DeterministicItemFields {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, id := range ids {
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.Uint(uint64(id))); err != nil {
			return err
		}
		if err := writeArbitraryValue(enc, f[id]); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndArray)
}

// UnmarshalJSONFrom accepts both the legacy object form and the compact pairs form.
func (f *ItemFields) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	kind := dec.PeekKind()
	switch kind {
	case 'n': // null
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		if tok.Kind() != 'n' {
			return &json.SemanticError{Err: errors.New("expected null"), JSONKind: tok.Kind()}
		}
		*f = nil
		return nil
	case '{':
		return f.decodeObject(dec)
	case '[':
		return f.decodePairsArray(dec)
	default:
		return &json.SemanticError{Err: errors.New("invalid ItemFields kind"), JSONKind: kind}
	}
}

func (f *ItemFields) decodeObject(dec *jsontext.Decoder) error {
	if *f == nil {
		*f = make(ItemFields)
	} else {
		for k := range *f {
			delete(*f, k)
		}
	}
	// consume '{'
	if tok, err := dec.ReadToken(); err != nil {
		return err
	} else if tok.Kind() != '{' {
		return &json.SemanticError{Err: errors.New("expected '{'"), JSONKind: tok.Kind()}
	}
	for {
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		if tok.Kind() == '}' {
			break
		}
		if tok.Kind() != '"' {
			return &json.SemanticError{Err: errors.New("expected string key"), JSONKind: tok.Kind()}
		}
		id64, err := strconv.ParseUint(tok.String(), 10, 64)
		if err != nil {
			return &json.SemanticError{Err: errors.New("non-numeric field id"), JSONKind: tok.Kind()}
		}
		val, err := readArbitraryValue(dec)
		if err != nil {
			return err
		}
		(*f)[uint(id64)] = val
	}
	return nil
}

func (f *ItemFields) decodePairsArray(dec *jsontext.Decoder) error {
	if *f == nil {
		*f = make(ItemFields)
	} else {
		for k := range *f {
			delete(*f, k)
		}
	}
	// consume outer '['
	if tok, err := dec.ReadToken(); err != nil {
		return err
	} else if tok.Kind() != '[' {
		return &json.SemanticError{Err: errors.New("expected '['"), JSONKind: tok.Kind()}
	}
	for {
		k := dec.PeekKind()
		if k == ']' {
			if _, err := dec.ReadToken(); err != nil {
				return err
			}
			break
		}
		// inner [id,value]
		if tok, err := dec.ReadToken(); err != nil {
			return err
		} else if tok.Kind() != '[' {
			return &json.SemanticError{Err: errors.New("expected '[' for pair"), JSONKind: tok.Kind()}
		}
		idTok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		if idTok.Kind() != '0' {
			return &json.SemanticError{Err: errors.New("expected numeric id"), JSONKind: idTok.Kind()}
		}
		id64, err := strconv.ParseUint(idTok.String(), 10, 64)
		if err != nil {
			return err
		}
		val, err := readArbitraryValue(dec)
		if err != nil {
			return err
		}
		endTok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		if endTok.Kind() != ']' {
			return &json.SemanticError{Err: errors.New("expected end of pair array"), JSONKind: endTok.Kind()}
		}
		(*f)[uint(id64)] = val
	}
	return nil
}

// --- helpers ---
func writeArbitraryValue(enc *jsontext.Encoder, v interface{}) error {
	switch x := v.(type) {
	case nil:
		return enc.WriteToken(jsontext.Null)
	case string:
		return enc.WriteToken(jsontext.String(x))
	case int:
		return enc.WriteToken(jsontext.Int(int64(x)))
	case int64:
		return enc.WriteToken(jsontext.Int(x))
	case uint:
		return enc.WriteToken(jsontext.Uint(uint64(x)))
	case float64:
		return enc.WriteToken(jsontext.Float(x))
	case bool:
		if x {
			return enc.WriteToken(jsontext.True)
		}
		return enc.WriteToken(jsontext.False)
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return err
		}
		return enc.WriteValue(jsontext.Value(b))
	}
}

func readArbitraryValue(dec *jsontext.Decoder) (interface{}, error) {
	kind := dec.PeekKind()
	switch kind {
	case 'n':
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		return nil, nil
	case '"':
		t, err := dec.ReadToken()
		if err != nil {
			return nil, err
		}
		return t.String(), nil
	case '0':
		t, err := dec.ReadToken()
		if err != nil {
			return nil, err
		}
		if i, err := strconv.ParseInt(t.String(), 10, 64); err == nil {
			if i >= -1<<31 && i <= (1<<31-1) {
				return int(i), nil
			}
			return i, nil
		}
		if f, err := strconv.ParseFloat(t.String(), 64); err == nil {
			return f, nil
		}
		return t.String(), nil
	case 't', 'f':
		t, err := dec.ReadToken()
		if err != nil {
			return nil, err
		}
		return t.Bool(), nil
	case '[':
		var any interface{}
		if err := json.UnmarshalDecode(dec, &any); err != nil {
			return nil, err
		}
		return any, nil
	case '{':
		var any interface{}
		if err := json.UnmarshalDecode(dec, &any); err != nil {
			return nil, err
		}
		return any, nil
	default:
		return nil, &json.SemanticError{Err: errors.New("unsupported JSON kind"), JSONKind: kind}
	}
}
