package types

import "io"

type FacetId uint32
type ItemId uint32

type Item interface {
	GetId() ItemId
	GetSku() string
	GetStock() map[string]uint16
	UpdateStock(locationId string, quantity uint16) error
	HasStock() bool
	IsDeleted() bool
	IsSoftDeleted() bool
	GetPropertyValue(name string) any
	GetPrice() int
	GetDiscount() int
	GetRating() (int, int)
	//GetFieldValue(id uint) (interface{}, bool)
	GetStringFields() map[FacetId]string
	GetNumberFields() map[FacetId]float64
	GetStringFieldValue(id FacetId) (string, bool)
	GetStringsFieldValue(id FacetId) ([]string, bool)
	GetNumberFieldValue(id FacetId) (float64, bool)

	GetLastUpdated() int64
	GetCreated() int64
	//GetPopularity() float64
	GetTitle() string
	ToString() string
	ToStringList() []string
	//GetBaseItem() BaseItem
	//MergeKeyFields(updates []CategoryUpdate) bool
	//	GetItem() interface{}
	CanHaveEmbeddings() bool
	GetEmbeddingsText() (string, error)
	Write(writer io.Writer) (int, error)
	//StreamLine(w http.ResponseWriter)
}
