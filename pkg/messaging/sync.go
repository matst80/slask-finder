package messaging

type ChangeTopic string

const (
	ItemsChanged  ChangeTopic = "item_changed"
	FacetsChanged ChangeTopic = "facets_changed"
)
