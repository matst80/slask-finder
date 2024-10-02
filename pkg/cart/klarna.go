package cart

import "time"

type CreateOrder struct {
	PurchaseCountry        string                   `json:"purchase_country"`
	PurchaseCurrency       string                   `json:"purchase_currency"`
	Locale                 string                   `json:"locale"`
	Status                 string                   `json:"status"`
	BillingAddress         BillingAddress           `json:"billing_address"`
	ShippingAddress        ShippingAddress          `json:"shipping_address"`
	OrderAmount            int                      `json:"order_amount"`
	OrderTaxAmount         int                      `json:"order_tax_amount"`
	OrderLines             []OrderLines             `json:"order_lines"`
	Customer               Customer                 `json:"customer"`
	MerchantUrls           MerchantUrls             `json:"merchant_urls"`
	MerchantReference1     string                   `json:"merchant_reference1"`
	MerchantReference2     string                   `json:"merchant_reference2"`
	Options                Options                  `json:"options"`
	Attachment             Attachment               `json:"attachment"`
	ExternalPaymentMethods []ExternalPaymentMethods `json:"external_payment_methods"`
	ExternalCheckouts      []ExternalCheckouts      `json:"external_checkouts"`
	ShippingCountries      []string                 `json:"shipping_countries"`
	ShippingOptions        []ShippingOptions        `json:"shipping_options"`
	MerchantData           string                   `json:"merchant_data"`
	Gui                    Gui                      `json:"gui"`
	MerchantRequested      MerchantRequested        `json:"merchant_requested"`
	SelectedShippingOption SelectedShippingOption   `json:"selected_shipping_option"`
	Recurring              bool                     `json:"recurring"`
	BillingCountries       []string                 `json:"billing_countries"`
	Tags                   []string                 `json:"tags"`
	DiscountLines          []DiscountLines          `json:"discount_lines"`
}
type BillingAddress struct {
	GivenName      string `json:"given_name"`
	FamilyName     string `json:"family_name"`
	Email          string `json:"email"`
	Title          string `json:"title"`
	StreetAddress  string `json:"street_address"`
	StreetAddress2 string `json:"street_address2"`
	StreetName     string `json:"street_name"`
	StreetNumber   string `json:"street_number"`
	HouseExtension string `json:"house_extension"`
	PostalCode     string `json:"postal_code"`
	City           string `json:"city"`
	Region         string `json:"region"`
	Phone          string `json:"phone"`
	Country        string `json:"country"`
	CareOf         string `json:"care_of"`
}
type ShippingAddress struct {
	GivenName      string `json:"given_name"`
	FamilyName     string `json:"family_name"`
	Email          string `json:"email"`
	Title          string `json:"title"`
	StreetAddress  string `json:"street_address"`
	StreetAddress2 string `json:"street_address2"`
	StreetName     string `json:"street_name"`
	StreetNumber   string `json:"street_number"`
	HouseExtension string `json:"house_extension"`
	PostalCode     string `json:"postal_code"`
	City           string `json:"city"`
	Region         string `json:"region"`
	Phone          string `json:"phone"`
	Country        string `json:"country"`
	CareOf         string `json:"care_of"`
}
type Subscription struct {
	Name          string `json:"name"`
	Interval      string `json:"interval"`
	IntervalCount int    `json:"interval_count"`
}
type ProductIdentifiers struct {
	Brand                  string `json:"brand"`
	Color                  string `json:"color"`
	CategoryPath           string `json:"category_path"`
	GlobalTradeItemNumber  string `json:"global_trade_item_number"`
	ManufacturerPartNumber string `json:"manufacturer_part_number"`
	Size                   string `json:"size"`
}
type Dimensions struct {
	Height int `json:"height"`
	Width  int `json:"width"`
	Length int `json:"length"`
}
type ShippingAttributes struct {
	Weight     int        `json:"weight"`
	Dimensions Dimensions `json:"dimensions"`
	Tags       []string   `json:"tags"`
}
type OrderLines struct {
	Type                string             `json:"type"`
	Reference           string             `json:"reference"`
	Name                string             `json:"name"`
	Quantity            int                `json:"quantity"`
	Subscription        Subscription       `json:"subscription"`
	QuantityUnit        string             `json:"quantity_unit"`
	UnitPrice           int                `json:"unit_price"`
	TaxRate             int                `json:"tax_rate"`
	TotalAmount         int                `json:"total_amount"`
	TotalDiscountAmount int                `json:"total_discount_amount"`
	TotalTaxAmount      int                `json:"total_tax_amount"`
	MerchantData        string             `json:"merchant_data"`
	ProductURL          string             `json:"product_url"`
	ImageURL            string             `json:"image_url"`
	ProductIdentifiers  ProductIdentifiers `json:"product_identifiers"`
	ShippingAttributes  ShippingAttributes `json:"shipping_attributes"`
}
type Customer struct {
	Type                       string `json:"type"`
	Gender                     string `json:"gender"`
	DateOfBirth                string `json:"date_of_birth"`
	OrganizationRegistrationID string `json:"organization_registration_id"`
	VatID                      string `json:"vat_id"`
}
type MerchantUrls struct {
	Terms                string `json:"terms"`
	Checkout             string `json:"checkout"`
	Confirmation         string `json:"confirmation"`
	Push                 string `json:"push"`
	Validation           string `json:"validation"`
	Notification         string `json:"notification"`
	CancellationTerms    string `json:"cancellation_terms"`
	ShippingOptionUpdate string `json:"shipping_option_update"`
	AddressUpdate        string `json:"address_update"`
	CountryChange        string `json:"country_change"`
}
type AdditionalCheckbox struct {
	Text     string `json:"text"`
	Checked  bool   `json:"checked"`
	Required bool   `json:"required"`
}
type AdditionalCheckboxes struct {
	Text     string `json:"text"`
	Checked  bool   `json:"checked"`
	Required bool   `json:"required"`
	ID       string `json:"id"`
}
type Options struct {
	RequireValidateCallbackSuccess        bool                   `json:"require_validate_callback_success"`
	AcquiringChannel                      string                 `json:"acquiring_channel"`
	VatRemoved                            bool                   `json:"vat_removed"`
	AllowSeparateShippingAddress          bool                   `json:"allow_separate_shipping_address"`
	ColorButton                           string                 `json:"color_button"`
	ColorButtonText                       string                 `json:"color_button_text"`
	ColorCheckbox                         string                 `json:"color_checkbox"`
	ColorCheckboxCheckmark                string                 `json:"color_checkbox_checkmark"`
	ColorHeader                           string                 `json:"color_header"`
	ColorLink                             string                 `json:"color_link"`
	DateOfBirthMandatory                  bool                   `json:"date_of_birth_mandatory"`
	ShippingDetails                       string                 `json:"shipping_details"`
	TitleMandatory                        bool                   `json:"title_mandatory"`
	AdditionalCheckbox                    AdditionalCheckbox     `json:"additional_checkbox"`
	NationalIdentificationNumberMandatory bool                   `json:"national_identification_number_mandatory"`
	AdditionalMerchantTerms               string                 `json:"additional_merchant_terms"`
	PhoneMandatory                        bool                   `json:"phone_mandatory"`
	RadiusBorder                          string                 `json:"radius_border"`
	AllowedCustomerTypes                  []string               `json:"allowed_customer_types"`
	ShowSubtotalDetail                    bool                   `json:"show_subtotal_detail"`
	AdditionalCheckboxes                  []AdditionalCheckboxes `json:"additional_checkboxes"`
	VerifyNationalIdentificationNumber    bool                   `json:"verify_national_identification_number"`
	AutoCapture                           bool                   `json:"auto_capture"`
	RequireClientValidation               bool                   `json:"require_client_validation"`
	EnableDiscountModule                  bool                   `json:"enable_discount_module"`
	ShowVatRegistrationNumberField        bool                   `json:"show_vat_registration_number_field"`
}
type Attachment struct {
	Body        time.Time `json:"body"`
	ContentType string    `json:"content_type"`
}
type ExternalPaymentMethods struct {
	Name        string   `json:"name"`
	Fee         int      `json:"fee"`
	Description string   `json:"description"`
	Countries   []string `json:"countries"`
	Label       string   `json:"label"`
	RedirectURL string   `json:"redirect_url"`
	ImageURL    string   `json:"image_url"`
}
type ExternalCheckouts struct {
	Name        string   `json:"name"`
	Fee         int      `json:"fee"`
	Description string   `json:"description"`
	Countries   []string `json:"countries"`
	Label       string   `json:"label"`
	RedirectURL string   `json:"redirect_url"`
	ImageURL    string   `json:"image_url"`
}
type Product struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}
type Timeslot struct {
	ID    string `json:"id"`
	Start string `json:"start"`
	End   string `json:"end"`
}
type Address struct {
	GivenName      string `json:"given_name"`
	FamilyName     string `json:"family_name"`
	Email          string `json:"email"`
	Title          string `json:"title"`
	StreetAddress  string `json:"street_address"`
	StreetAddress2 string `json:"street_address2"`
	StreetName     string `json:"street_name"`
	StreetNumber   string `json:"street_number"`
	HouseExtension string `json:"house_extension"`
	PostalCode     string `json:"postal_code"`
	City           string `json:"city"`
	Region         string `json:"region"`
	Phone          string `json:"phone"`
	Country        string `json:"country"`
	CareOf         string `json:"care_of"`
}
type PickupLocation struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Address Address `json:"address"`
}
type DeliveryDetails struct {
	Carrier        string         `json:"carrier"`
	Class          string         `json:"class"`
	Product        Product        `json:"product"`
	Timeslot       Timeslot       `json:"timeslot"`
	PickupLocation PickupLocation `json:"pickup_location"`
}
type SelectedAddons struct {
	Type       string `json:"type"`
	Price      int    `json:"price"`
	ExternalID string `json:"external_id"`
	UserInput  string `json:"user_input"`
}
type ShippingOptions struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Promo           string           `json:"promo"`
	Price           int              `json:"price"`
	Preselected     bool             `json:"preselected"`
	TaxAmount       int              `json:"tax_amount"`
	TaxRate         int              `json:"tax_rate"`
	ShippingMethod  string           `json:"shipping_method"`
	DeliveryDetails DeliveryDetails  `json:"delivery_details"`
	TmsReference    string           `json:"tms_reference"`
	SelectedAddons  []SelectedAddons `json:"selected_addons"`
}
type Gui struct {
	Options []string `json:"options"`
}
type MerchantRequested struct {
}
type SelectedShippingOption struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Promo           string           `json:"promo"`
	Price           int              `json:"price"`
	Preselected     bool             `json:"preselected"`
	TaxAmount       int              `json:"tax_amount"`
	TaxRate         int              `json:"tax_rate"`
	ShippingMethod  string           `json:"shipping_method"`
	DeliveryDetails DeliveryDetails  `json:"delivery_details"`
	TmsReference    string           `json:"tms_reference"`
	SelectedAddons  []SelectedAddons `json:"selected_addons"`
}
type DiscountLines struct {
	Name           string `json:"name"`
	Quantity       int    `json:"quantity"`
	UnitPrice      int    `json:"unit_price"`
	TaxRate        int    `json:"tax_rate"`
	TotalAmount    int    `json:"total_amount"`
	TotalTaxAmount int    `json:"total_tax_amount"`
	Reference      string `json:"reference"`
	MerchantData   string `json:"merchant_data"`
}

func MakeCreateOrderFromCart(cart *Cart) CreateOrder {
	lines := make([]OrderLines, len(cart.Items))
	for i, item := range cart.Items {
		lines[i] = OrderLines{
			Type:                "physical",
			Reference:           item.Sku,
			Name:                item.Title,
			Quantity:            int(item.Quantity),
			UnitPrice:           item.Price,
			TotalAmount:         item.Price * int(item.Quantity),
			TotalDiscountAmount: item.OriginalPrice - item.Price,
			TaxRate:             25,
			ImageURL:            "https://elgiganten.se" + item.ImageUrl,
			TotalTaxAmount:      item.TaxAmount,
		}
	}
	return CreateOrder{
		PurchaseCountry:  "SE",
		PurchaseCurrency: "SEK",
		Locale:           "sv-SE",
		Status:           "checkout_incomplete",
		BillingAddress: BillingAddress{
			Country: "SE",
		},
		ShippingAddress: ShippingAddress{
			Country: "SE",
		},
		OrderAmount:    cart.TotalPrice,
		OrderTaxAmount: cart.TaxAmount,
		Customer: Customer{
			Type: "person",
		},
		MerchantUrls: MerchantUrls{
			Terms:        "https://example.com/terms",
			Checkout:     "https://example.com/checkout",
			Confirmation: "https://example.com/confirmation",
			Push:         "https://slask-finder.tornberg.me/api/push",
		},
		Options: Options{
			RequireValidateCallbackSuccess: true,
		},
		OrderLines: lines,
	}
}
