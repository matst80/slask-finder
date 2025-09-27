package main

type SuggestResult struct {
	Prefix string   `json:"prefix"`
	Word   string   `json:"match"`
	Other  []string `json:"other"`
	Hits   int      `json:"hits"`
}
