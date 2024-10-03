package cart

import (
	"fmt"
	"time"
)

type OrderLine struct {
	Type                string `json:"type"`
	Reference           string `json:"reference"`
	Name                string `json:"name"`
	Quantity            int    `json:"quantity"`
	QuantityUnit        string `json:"quantity_unit"`
	UnitPrice           int    `json:"unit_price"`
	TaxRate             int    `json:"tax_rate"`
	TotalAmount         int    `json:"total_amount"`
	TotalDiscountAmount int    `json:"total_discount_amount"`
	TotalTaxAmount      int    `json:"total_tax_amount"`
}

type OrderAddress struct {
	Country string `json:"country"`
}

type OrderMerchantUrls struct {
	Terms        string `json:"terms"`
	Checkout     string `json:"checkout"`
	Confirmation string `json:"confirmation"`
	Push         string `json:"push"`
}

type OrderOptions struct {
	AllowSeparateShippingAddress   bool `json:"allow_separate_shipping_address"`
	DateOfBirthMandatory           bool `json:"date_of_birth_mandatory"`
	RequireValidateCallbackSuccess bool `json:"require_validate_callback_success"`
}

type CreateOrderRequest struct {
	OrderID          string       `json:"order_id"`
	Status           string       `json:"status"`
	PurchaseCountry  string       `json:"purchase_country"`
	PurchaseCurrency string       `json:"purchase_currency"`
	Locale           string       `json:"locale"`
	BillingAddress   OrderAddress `json:"billing_address"`
	Customer         struct {
	} `json:"customer"`
	ShippingAddress        OrderAddress      `json:"shipping_address"`
	OrderAmount            int               `json:"order_amount"`
	OrderTaxAmount         int               `json:"order_tax_amount"`
	OrderLines             []OrderLine       `json:"order_lines"`
	MerchantUrls           OrderMerchantUrls `json:"merchant_urls"`
	HTMLSnippet            string            `json:"html_snippet"`
	StartedAt              time.Time         `json:"started_at"`
	LastModifiedAt         time.Time         `json:"last_modified_at"`
	Options                OrderOptions      `json:"options"`
	ExternalPaymentMethods []any             `json:"external_payment_methods"`
	ExternalCheckouts      []any             `json:"external_checkouts"`
}

func OrderRequestFromCart(cart *Cart) CreateOrderRequest {
	lines := make([]OrderLine, len(cart.Items))
	for i, item := range cart.Items {
		lines[i] = OrderLine{
			Type:         "physical",
			Reference:    item.Sku,
			Name:         item.Title,
			Quantity:     int(item.Quantity),
			QuantityUnit: "pcs",
			UnitPrice:    item.Price,
			//TaxRate:             int(item.TaxRate * 100),
			TotalAmount:         item.Price * int(item.Quantity),
			TotalDiscountAmount: 0,
			TotalTaxAmount:      0, // TODO Fix this
		}
	}
	return CreateOrderRequest{
		OrderID:          fmt.Sprintf("%d", cart.Id),
		Status:           "created",
		PurchaseCountry:  "se",
		PurchaseCurrency: "SEK",
		Locale:           "sv-se",
		BillingAddress: OrderAddress{
			Country: "se",
		},
		Customer: struct {
		}{},
		ShippingAddress: OrderAddress{
			Country: "se",
		},
		OrderAmount:    cart.TotalPrice,
		OrderTaxAmount: int(float64(cart.TotalPrice) * 0.25),
		OrderLines:     lines,
	}
}
