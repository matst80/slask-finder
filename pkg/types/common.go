package types

import "slices"

type BaseField struct {
	Id               FacetId `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description,omitempty"`
	Priority         float64 `json:"prio,omitempty"`
	Type             string  `json:"valueType,omitempty"`
	LinkedId         FacetId `json:"linkedId,omitempty"`
	ValueSorting     uint    `json:"sorting,omitempty"`
	GroupId          uint    `json:"groupId,omitempty"`
	CategoryLevel    int     `json:"categoryLevel,omitempty"`
	HideFacet        bool    `json:"hide,omitempty"`
	KeySpecification bool    `json:"isKey,omitempty"`
	InternalOnly     bool    `json:"internal,omitempty"`
	Searchable       bool    `json:"searchable,omitempty"`
	// IgnoreCategoryIfSearched bool    `json:"-"`
	// IgnoreIfInSearch         bool    `json:"-"`
}

type FacetRequest struct {
	*Filters
	Query        string    `json:"query" schema:"query"`
	Stock        []string  `json:"stock" schema:"stock"`
	IgnoreFacets []FacetId `json:"skipFacets" schema:"sf"`
}

func (s *FacetRequest) Sanitize() {
	if (len(s.StringFilter) > 0 || len(s.RangeFilter) > 0) && s.Query == "*" {
		s.Query = ""
	}
}

func (b *BaseField) UpdateFrom(field *BaseField) {
	if field == nil {
		return
	}
	//b.Id = field.Id
	if field.Name != "" {
		b.Name = field.Name
	}
	if field.Description != "" {
		b.Description = field.Description
	}

	b.Priority = field.Priority
	if field.Type != "" {
		b.Type = field.Type
	}
	b.LinkedId = field.LinkedId
	b.ValueSorting = field.ValueSorting
	b.Searchable = field.Searchable
	b.HideFacet = field.HideFacet
	b.CategoryLevel = field.CategoryLevel
	b.GroupId = field.GroupId
	b.KeySpecification = field.KeySpecification
	b.InternalOnly = field.InternalOnly
}

func (f *FacetRequest) HasField(id FacetId) bool {
	for _, v := range f.StringFilter {
		if v.Id == id {
			return true
		}
	}
	for _, v := range f.RangeFilter {
		if v.Id == id {
			return true
		}
	}
	return false
}

func (f *FacetRequest) IsIgnored(id FacetId) bool {
	// should be config
	if id >= 11 && id <= 14 {
		for _, sf := range f.StringFilter {
			if sf.Id > 9 && sf.Id < 14 && sf.Id != id {
				return false
			}
		}
		return true
	}
	return slices.Contains(f.IgnoreFacets, id)
}

type LocationStock []struct {
	Id    string `json:"id"`
	Level string `json:"level"`
}

type BaseItem struct {
	Id    ItemId
	Sku   string
	Title string
	Price int
	Img   string
}

type CategoryUpdate struct {
	Id    ItemId `json:"id"`
	Value string `json:"value"`
}

const FacetKeyType = 1
const FacetNumberType = 2
const FacetIntegerType = 3
const FacetTreeType = 4

type Embeddings []float32

type EmbeddingsEngine interface {
	GenerateEmbeddings(text string) (Embeddings, error)
	//GenerateEmbeddingsFromItem(item Item) (Embeddings, error)
}

type Facet interface {
	GetType() uint
	Match(data any) *ItemList
	// MatchAsync(data interface{}, results chan<- *ItemList)
	GetBaseField() *BaseField
	AddValueLink(value any, id ItemId) bool
	RemoveValueLink(value any, id ItemId)
	UpdateBaseField(data *BaseField)
	GetValues() []any
	IsExcludedFromFacets() bool
	IsCategory() bool
}

type FieldChangeAction = string

const (
	ADD_FIELD    FieldChangeAction = "add"
	REMOVE_FIELD FieldChangeAction = "remove"
	UPDATE_FIELD FieldChangeAction = "update"
)

type FieldChange struct {
	*BaseField
	Action    FieldChangeAction `json:"action"`
	FieldType uint              `json:"fieldType"`
}

type SettingsKey string

type SettingsChange struct {
	Type     SettingsKey `json:"type"`
	Priority float64     `json:"priority"`
	Value    any         `json:"value"`
}

func (s *Settings) Lock() {
	s.mu.Lock()
}
func (s *Settings) Unlock() {
	s.mu.Unlock()
}
func (s *Settings) RLock() {
	s.mu.RLock()
}
func (s *Settings) RUnlock() {
	s.mu.RUnlock()
}
