package main

type PostalCodeLocation struct {
	PostalCode string   `json:"postalCode"`
	City       string   `json:"city"`
	Location   Location `json:"location"`
}
