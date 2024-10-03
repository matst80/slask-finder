package promotions

import "fmt"

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
	MaxAmount   uint               `json:"max_amount"`
	Articles    []PromotionArticle `json:"articles"`
}

type PromotionOutput struct {
	*PromotionInput
	PromotionId int `json:"promotion_id"`
	Discount    int `json:"discount"`
}

type PromotionInput struct {
	Id       int    `json:"id"`
	Sku      string `json:"sku"`
	Quantity uint   `json:"qty"`
	Price    int    `json:"price"`
}

func (p *Promotion) IsAvailable(input ...*PromotionInput) int {
	couldBeApplied := 10000
	for _, article := range p.Articles {
		matches := 0
		for _, i := range input {
			if i.Sku == article.Sku {
				matches += int(i.Quantity)
			}
		}
		couldBeApplied = min(couldBeApplied, matches)
	}

	return couldBeApplied
}

func (a *PromotionAction) Apply(input *PromotionInput) (PromotionOutput, error) {
	result := PromotionOutput{
		PromotionInput: input,
		Discount:       0,
	}
	switch a.Type {
	case "percentage":
		result.Discount = int(float64(input.Price) * float64(a.Value/100))
	case "fixed":
		result.Discount = int(a.Value)
	default:
		return result, fmt.Errorf("unknown action type %s", a.Type)
	}
	return result, nil
}

func (p *Promotion) Apply(current *PromotionInput, others ...*PromotionInput) (*[]PromotionOutput, error) {
	all := append(others, current)
	if p.IsAvailable(all...) < 1 {
		return nil, fmt.Errorf("promotion not available")
	}
	result := make([]PromotionOutput, 0)
	for _, article := range p.Articles {
		for _, i := range all {
			if i.Sku == article.Sku {
				for _, action := range article.Actions {
					output, err := action.Apply(i)
					if err != nil {
						return nil, err
					}
					output.PromotionId = p.Id
					result = append(result, output)
				}
				break
			}
		}
	}
	return &result, nil
}
