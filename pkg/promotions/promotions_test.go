package promotions

import (
	"testing"
)

func TestPromotion(t *testing.T) {
	promotion := Promotion{
		Id:          1,
		Name:        "Promotion 1",
		Description: "Description 1",
		Articles: []PromotionArticle{
			{
				Sku: "sku1",
				Actions: []PromotionAction{
					{
						Type:  "percentage",
						Value: 1.0,
					},
				},
			},
		},
	}
	if promotion.Id != 1 {
		t.Errorf("Expected 1")
	}
}
