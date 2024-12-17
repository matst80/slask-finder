package types

import (
	"sync"
)

type OutOfStockRule struct {
	NoStoreMultiplier float64 `json:"noStoreMultiplier"`
	NoStockValue      float64 `json:"noStockValue"`
}

func (_ *OutOfStockRule) Type() RuleType {
	return "OutOfStockRule"
}

func (_ *OutOfStockRule) New() JsonType {
	return &OutOfStockRule{}
}

func (r *OutOfStockRule) GetValue(item Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	stores := len(item.GetStock())
	if stores > 0 {
		res <- float64(stores) * r.NoStoreMultiplier
		return
	}
	level := GetPropertyValue(item, "StockLevel")
	hasStock := false
	switch l := level.(type) {
	case string:
		hasStock = l != "" && l != "0"
	}
	if hasStock {
		res <- 0
	} else {
		res <- r.NoStockValue
	}
}
