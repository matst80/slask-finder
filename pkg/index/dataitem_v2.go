//go:build jsonv2

package index

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"strconv"
)

// MarshalJSONTo provides a custom low-allocation encoding path for DataItem when
// using encoding/json/v2. It deliberately inlines only the hot fields and then
// delegates the complex "values" map to the ItemFields custom encoder.
// Fields that are zero-value and tagged with omitempty are skipped similar to
// the standard library's behavior.
func (d *DataItem) aMarshalJSONTo(enc *jsontext.Encoder) error {
	if d == nil {
		return enc.WriteToken(jsontext.Null)
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}

	writeString := func(key, val string) error {
		if val == "" {
			return nil
		}
		if err := enc.WriteToken(jsontext.String(key)); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.String(val))
	}
	writeBool := func(key string, val bool) error {
		// no omitempty on buyable/buyableInStore so always emit
		if err := enc.WriteToken(jsontext.String(key)); err != nil {
			return err
		}
		if val {
			return enc.WriteToken(jsontext.True)
		}
		return enc.WriteToken(jsontext.False)
	}
	writeInt := func(key string, v int) error {
		if err := enc.WriteToken(jsontext.String(key)); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.Int(int64(v)))
	}
	writeInt64Omit := func(key string, v int64) error {
		if v == 0 {
			return nil
		}
		if err := enc.WriteToken(jsontext.String(key)); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.Int(v))
	}

	// Required / always present subset
	if err := writeInt("id", int(d.Id)); err != nil {
		return err
	}
	if err := writeString("sku", d.Sku); err != nil {
		return err
	}
	if err := writeString("title", d.Title); err != nil {
		return err
	}

	// Stock (map[string]string) - always emit (empty map if nil)
	if err := enc.WriteToken(jsontext.String("stock")); err != nil {
		return err
	}
	if d.Stock == nil {
		d.Stock = map[string]string{}
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for k, v := range d.Stock {
		if err := enc.WriteToken(jsontext.String(k)); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(v)); err != nil {
			return err
		}
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return err
	}

	// ItemProp selective emission (respecting omitempty)
	ip := &d.BaseItem.ItemProp
	_ = writeString // ensure referenced
	if ip.Url != "" {
		if err := writeString("url", ip.Url); err != nil {
			return err
		}
	}
	if ip.Disclaimer != "" {
		if err := writeString("disclaimer", ip.Disclaimer); err != nil {
			return err
		}
	}
	if ip.ReleaseDate != "" {
		if err := writeString("releaseDate", ip.ReleaseDate); err != nil {
			return err
		}
	}
	if ip.SaleStatus != "" {
		if err := writeString("saleStatus", ip.SaleStatus); err != nil {
			return err
		}
	}
	if ip.OnlineSaleStatus != "" {
		if err := writeString("onlineSaleStatus", ip.OnlineSaleStatus); err != nil {
			return err
		}
	}
	if ip.PresaleDate != "" {
		if err := writeString("presaleDate", ip.PresaleDate); err != nil {
			return err
		}
	}
	if ip.Restock != "" {
		if err := writeString("restock", ip.Restock); err != nil {
			return err
		}
	}
	if ip.AdvertisingText != "" {
		if err := writeString("advertisingText", ip.AdvertisingText); err != nil {
			return err
		}
	}
	if ip.Img != "" {
		if err := writeString("img", ip.Img); err != nil {
			return err
		}
	}
	if ip.BadgeUrl != "" {
		if err := writeString("badgeUrl", ip.BadgeUrl); err != nil {
			return err
		}
	}
	if ip.BulletPoints != "" {
		if err := writeString("bp", ip.BulletPoints); err != nil {
			return err
		}
	}
	if err := writeInt64Omit("lastUpdate", ip.LastUpdate); err != nil {
		return err
	}
	if err := writeInt64Omit("created", ip.Created); err != nil {
		return err
	}
	if ip.Description != "" {
		if err := writeString("description", ip.Description); err != nil {
			return err
		}
	}
	if ip.BoxSize != "" {
		if err := writeString("boxSize", ip.BoxSize); err != nil {
			return err
		}
	}
	if ip.ArticleType != "" {
		if err := writeString("articleType", ip.ArticleType); err != nil {
			return err
		}
	}

	// Always emit buyable flags (no omitempty in original struct tags)
	if err := writeBool("buyable", ip.Buyable); err != nil {
		return err
	}
	if err := writeBool("buyableInStore", ip.BuyableInStore); err != nil {
		return err
	}

	// MarginPercent custom rule (existing MarshalJSON logic) -> either 0.0 or real
	if err := enc.WriteToken(jsontext.String("mp")); err != nil {
		return err
	}
	if AllowConditionalData {
		if err := enc.WriteToken(jsontext.Float(float64(ip.MarginPercent))); err != nil {
			return err
		}
	} else {
		if err := enc.WriteToken(jsontext.Float(0.0)); err != nil {
			return err
		}
	}

	// Optional nested structs
	if ip.CheapestBItem != nil {
		if err := enc.WriteToken(jsontext.String("bItem")); err != nil {
			return err
		}
		if err := marshalOutletItem(enc, ip.CheapestBItem); err != nil {
			return err
		}
	}
	if ip.AItem != nil {
		if err := enc.WriteToken(jsontext.String("aItem")); err != nil {
			return err
		}
		if err := marshalOutletItem(enc, ip.AItem); err != nil {
			return err
		}
	}
	if ip.EnergyRating != nil {
		if err := enc.WriteToken(jsontext.String("energyRating")); err != nil {
			return err
		}
		if err := marshalEnergy(enc, ip.EnergyRating); err != nil {
			return err
		}
	}

	// values (ItemFields) -> delegate
	if err := enc.WriteToken(jsontext.String("values")); err != nil {
		return err
	}
	if err := d.Fields.MarshalJSONTo(enc); err != nil {
		return err
	}

	return enc.WriteToken(jsontext.EndObject)
}

func marshalOutletItem(enc *jsontext.Encoder, o *OutletItem) error {
	if o == nil {
		return enc.WriteToken(jsontext.Null)
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if o.ArticleNumber != "" {
		if err := enc.WriteToken(jsontext.String("sku")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(o.ArticleNumber)); err != nil {
			return err
		}
	}
	if err := enc.WriteToken(jsontext.String("price")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("inc")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Int(int64(o.Price.IncVat))); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("exl")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Int(int64(o.Price.ExVat))); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("title")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(o.Title)); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

func marshalEnergy(enc *jsontext.Encoder, e *EnergyRating) error {
	if e == nil {
		return enc.WriteToken(jsontext.Null)
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if e.Value != "" {
		if err := enc.WriteToken(jsontext.String("value")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(e.Value)); err != nil {
			return err
		}
	}
	if e.Min != "" {
		if err := enc.WriteToken(jsontext.String("min")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(e.Min)); err != nil {
			return err
		}
	}
	if e.Max != "" {
		if err := enc.WriteToken(jsontext.String("max")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(e.Max)); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSONFrom keeps correctness while still leveraging existing ItemFields decoder.
func (d *DataItem) aUnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() == 'n' {
		*d = DataItem{}
		return nil
	}
	if tok.Kind() != '{' {
		return &json.SemanticError{Err: errors.New("expected object for DataItem"), JSONKind: tok.Kind()}
	}
	if d.BaseItem == nil {
		d.BaseItem = &BaseItem{}
	}
	if d.BaseItem.Stock == nil {
		d.BaseItem.Stock = map[string]string{}
	}
	ip := &d.BaseItem.ItemProp
	for {
		k := dec.PeekKind()
		if k == '}' {
			_, _ = dec.ReadToken()
			break
		}
		keyTok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		if keyTok.Kind() != '"' {
			return &json.SemanticError{Err: errors.New("expected string key"), JSONKind: keyTok.Kind()}
		}
		key := keyTok.String()
		switch key {
		case "id":
			vTok, _ := dec.ReadToken()
			if vTok.Kind() == '0' {
				if id64, e := strconv.ParseUint(vTok.String(), 10, 64); e == nil {
					d.Id = uint(id64)
				}
			}
		case "sku":
			vTok, _ := dec.ReadToken()
			d.Sku = vTok.String()
		case "title":
			vTok, _ := dec.ReadToken()
			d.Title = vTok.String()
		case "stock":
			// parse object
			stTok, _ := dec.ReadToken()
			if stTok.Kind() == '{' {
				d.BaseItem.Stock = map[string]string{}
				for {
					pk := dec.PeekKind()
					if pk == '}' {
						_, _ = dec.ReadToken()
						break
					}
					sk, _ := dec.ReadToken()
					sv, _ := dec.ReadToken()
					d.BaseItem.Stock[sk.String()] = sv.String()
				}
			}
		case "url":
			vTok, _ := dec.ReadToken()
			ip.Url = vTok.String()
		case "disclaimer":
			vTok, _ := dec.ReadToken()
			ip.Disclaimer = vTok.String()
		case "releaseDate":
			vTok, _ := dec.ReadToken()
			ip.ReleaseDate = vTok.String()
		case "saleStatus":
			vTok, _ := dec.ReadToken()
			ip.SaleStatus = vTok.String()
		case "onlineSaleStatus":
			vTok, _ := dec.ReadToken()
			ip.OnlineSaleStatus = vTok.String()
		case "presaleDate":
			vTok, _ := dec.ReadToken()
			ip.PresaleDate = vTok.String()
		case "restock":
			vTok, _ := dec.ReadToken()
			ip.Restock = vTok.String()
		case "advertisingText":
			vTok, _ := dec.ReadToken()
			ip.AdvertisingText = vTok.String()
		case "img":
			vTok, _ := dec.ReadToken()
			ip.Img = vTok.String()
		case "badgeUrl":
			vTok, _ := dec.ReadToken()
			ip.BadgeUrl = vTok.String()
		case "bp":
			vTok, _ := dec.ReadToken()
			ip.BulletPoints = vTok.String()
		case "lastUpdate":
			vTok, _ := dec.ReadToken()
			if vTok.Kind() == '0' {
				if i, e := strconv.ParseInt(vTok.String(), 10, 64); e == nil {
					ip.LastUpdate = i
				}
			}
		case "created":
			vTok, _ := dec.ReadToken()
			if vTok.Kind() == '0' {
				if i, e := strconv.ParseInt(vTok.String(), 10, 64); e == nil {
					ip.Created = i
				}
			}
		case "description":
			vTok, _ := dec.ReadToken()
			ip.Description = vTok.String()
		case "boxSize":
			vTok, _ := dec.ReadToken()
			ip.BoxSize = vTok.String()
		case "articleType":
			vTok, _ := dec.ReadToken()
			ip.ArticleType = vTok.String()
		case "buyable":
			vTok, _ := dec.ReadToken()
			ip.Buyable = (vTok.Kind() == 't')
		case "buyableInStore":
			vTok, _ := dec.ReadToken()
			ip.BuyableInStore = (vTok.Kind() == 't')
		case "mp":
			vTok, _ := dec.ReadToken()
			if vTok.Kind() == '0' {
				if f64, e := strconv.ParseFloat(vTok.String(), 64); e == nil {
					ip.MarginPercent = MarginPercent(f64)
				}
			}
		case "bItem":
			// delegate to generic decode
			var oi OutletItem
			if err := json.UnmarshalDecode(dec, &oi); err != nil {
				return err
			}
			ip.CheapestBItem = &oi
		case "aItem":
			var oi OutletItem
			if err := json.UnmarshalDecode(dec, &oi); err != nil {
				return err
			}
			ip.AItem = &oi
		case "energyRating":
			var er EnergyRating
			if err := json.UnmarshalDecode(dec, &er); err != nil {
				return err
			}
			ip.EnergyRating = &er
		case "values":
			if err := d.Fields.UnmarshalJSONFrom(dec); err != nil {
				return err
			}
		default:
			// Skip a single JSON value (object/array recursively)
			if err := skipValue(dec); err != nil {
				return err
			}
		}
	}
	return nil
}

// skipValue consumes one JSON value from decoder.
func skipValue(dec *jsontext.Decoder) error {
	k := dec.PeekKind()
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	switch k {
	case '{':
		depth := 1
		for depth > 0 {
			t, err := dec.ReadToken()
			if err != nil {
				return err
			}
			switch t.Kind() {
			case '{':
				depth++
			case '}':
				depth--
			}
		}
	case '[':
		depth := 1
		for depth > 0 {
			t, err := dec.ReadToken()
			if err != nil {
				return err
			}
			switch t.Kind() {
			case '[':
				depth++
			case ']':
				depth--
			}
		}
	default:
		// primitives already consumed by initial ReadToken
		_ = tok
	}
	return nil
}
