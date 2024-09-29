package promotions

type PromotionAction struct {
	Market string  `json:"market"`
	Type   string  `json:"type"`
	Value  float64 `json:"value"`
}

type PromotionArticle struct {
	Sku     string            `json:"sku"`
	Actions []PromotionAction `json:"actions"`
}

type Promotion struct {
	Id          int                `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Articles    []PromotionArticle `json:"articles"`
}

type PromotionInput struct {
	Id       uint `json:"id"`
	Quantity uint `json:"quantity"`
}
