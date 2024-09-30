package promotions

import (
	"testing"
)

func TestPromotionAvailable(t *testing.T) {
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
	validInput := &PromotionInput{
		Sku:      "sku1",
		Quantity: 1,
		Price:    100,
	}
	if !promotion.IsAvailable(validInput) {
		t.Errorf("Expected to be available")
	}
	output, err := promotion.Apply(validInput)
	if err != nil {
		t.Errorf(err.Error())
	}
	if (*output)[0].Discount != 1 {
		t.Errorf("Expected discount to be 1, got %d", (*output)[0].Discount)
	}

	if promotion.IsAvailable(&PromotionInput{
		Sku:      "sku2",
		Quantity: 2,
	}) {
		t.Errorf("Expected to not be available")
	}

}

func TestPromotionMultipleBundleAvailable(t *testing.T) {
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
			{
				Sku: "sku2",
				Actions: []PromotionAction{
					{
						Type:  "percentage",
						Value: 1.0,
					},
				},
			},
		},
	}
	if promotion.IsAvailable(&PromotionInput{
		Sku:      "sku1",
		Quantity: 1,
	}) {
		t.Errorf("Expected to not be available")
	}

	if promotion.IsAvailable(&PromotionInput{
		Sku:      "sku2",
		Quantity: 2,
	}) != false {
		t.Errorf("Expected to not be available")
	}

	if !promotion.IsAvailable(&PromotionInput{
		Sku:      "sku2",
		Quantity: 2,
	}, &PromotionInput{
		Sku:      "sku1",
		Quantity: 1,
	}) != false {
		t.Errorf("Expected to be available")
	}
}
