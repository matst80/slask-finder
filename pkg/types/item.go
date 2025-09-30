package types

import "io"

type Item interface {
	GetId() uint
	GetSku() string
	GetStock() map[string]string
	HasStock() bool
	IsDeleted() bool
	IsSoftDeleted() bool
	GetPropertyValue(name string) any
	GetPrice() int
	GetDiscount() int
	GetRating() (int, int)
	//GetFieldValue(id uint) (interface{}, bool)
	GetStringFields() map[uint]string
	GetNumberFields() map[uint]float64
	GetStringFieldValue(id uint) (string, bool)
	GetStringsFieldValue(id uint) ([]string, bool)
	GetNumberFieldValue(id uint) (float64, bool)

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
