package types

type OutOfStockRule struct {
	NoStoreMultiplier float64 `json:"noStoreMultiplier"`
	NoStockValue      float64 `json:"noStockValue"`
}

func (r *OutOfStockRule) Type() RuleType {
	return "OutOfStockRule"
}

func (r *OutOfStockRule) New() JsonType {
	return &OutOfStockRule{}
}

func (r *OutOfStockRule) GetValue(item Item) float64 {

	stores := len(item.GetStock())
	if stores > 0 {
		return float64(stores) * r.NoStoreMultiplier
	}
	hasStock := item.HasStock() // GetPropertyValue(item, "StockLevel")

	if hasStock {
		return 0
	}
	return r.NoStockValue

}
