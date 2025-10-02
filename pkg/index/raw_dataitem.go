package index

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

type RawDataItem struct {
	mu         sync.RWMutex
	Id         uint
	Data       []byte
	cache      *DataItem // decoded object
	lastAccess time.Time
}

func NewRawDataItem(id uint, data []byte) *RawDataItem {
	return &RawDataItem{
		Id:         id,
		Data:       data,
		lastAccess: time.Now(),
	}
}

func NewRawConverted(item *DataItem) *RawDataItem {
	bytes, err := json.Marshal(item)
	if err != nil {
		panic(err)
	}
	return &RawDataItem{
		Id:         item.Id,
		Data:       bytes,
		cache:      item, // you control ownership: decide if this should be a pooled object or unique
		lastAccess: time.Now(),
	}
}

func (r *RawDataItem) GetId() uint {
	return r.Id
}

func (r *RawDataItem) Evict() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = nil
}

func (r *RawDataItem) LastAccess() time.Time {
	r.mu.RLock()
	ts := r.lastAccess
	r.mu.RUnlock()
	return ts
}

func (r *RawDataItem) getItem() *DataItem {

	// Fast read path
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastAccess = time.Now()
	cached := r.cache
	if cached != nil {
		return cached
	}

	// Slow path: need to decode
	if err := json.Unmarshal(r.Data, &r.cache); err != nil {
		panic(err)
	}

	return r.cache
}

// Below are pass-through wrappers:

func (item *RawDataItem) GetSku() string {
	return item.getItem().GetSku()
}

func (rawItem *RawDataItem) IsDeleted() bool {
	return rawItem.getItem().IsDeleted()
}

func (item *RawDataItem) HasStock() bool {
	return item.getItem().Buyable
}

func (item *RawDataItem) GetPropertyValue(name string) any {
	return item.getItem().GetPropertyValue(name)
}

func (rawItem *RawDataItem) IsSoftDeleted() bool {
	return rawItem.getItem().IsSoftDeleted()
}

func (item *RawDataItem) GetPrice() int {
	return item.getItem().GetPrice()
}

func (item *RawDataItem) GetStock() map[string]uint {
	return item.getItem().GetStock()
}

func (item *RawDataItem) GetStringFields() map[uint]string {
	return item.getItem().GetStringFields()
}

func (item *RawDataItem) GetNumberFields() map[uint]float64 {
	return item.getItem().GetNumberFields()
}

func (item *RawDataItem) GetStringFieldValue(id uint) (string, bool) {
	return item.getItem().GetStringFieldValue(id)
}

func (item *RawDataItem) GetStringsFieldValue(id uint) ([]string, bool) {
	return item.getItem().GetStringsFieldValue(id)
}

func (item *RawDataItem) GetNumberFieldValue(id uint) (float64, bool) {
	return item.getItem().GetNumberFieldValue(id)
}

func (item *RawDataItem) GetRating() (int, int) {
	return item.getItem().GetRating()
}

func (item *RawDataItem) CanHaveEmbeddings() bool {
	return item.getItem().CanHaveEmbeddings()
}

func (item *RawDataItem) GetEmbeddingsText() (string, error) {
	return item.getItem().GetEmbeddingsText()
}

func (item *RawDataItem) GetLastUpdated() int64 {
	return item.getItem().GetLastUpdated()
}

func (item *RawDataItem) GetCreated() int64 {
	return item.getItem().GetCreated()
}

func (item *RawDataItem) GetDiscount() int {
	return item.getItem().GetDiscount()
}

func (item *RawDataItem) GetTitle() string {
	return item.getItem().GetTitle()
}

func (item *RawDataItem) ToStringList() []string {
	return item.getItem().ToStringList()
}

func (item *RawDataItem) ToString() string {
	return item.getItem().ToString()
}

func (item *RawDataItem) Write(w io.Writer) (int, error) {
	b, err := w.Write(item.Data)
	if err != nil {
		return b, err
	}
	n, err := w.Write([]byte("\n"))
	return b + n, err
}
