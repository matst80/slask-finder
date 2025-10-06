package main

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
)

// getFieldType safely converts an internal int field type to uint while:
// 1. Preventing negative values (which would underflow to huge uints)
// 2. Validating against the set of supported facet field types (1=key,2=decimal,3=integer)
// 3. Avoiding triggering gosec G115 (int -> uint potential overflow) by explicit checks
// Accept either facet.FieldType or its underlying int (DataType) for flexibility
// T can be an int-like or uint-like underlying type (writer DataType likely int, facet.FieldType is uint)
func getFieldType[T ~int | ~uint](v T) (uint, bool) {
	if v < 0 { // negative would wrap
		return 0, false
	}
	switch v { // enumerate allowed types; adjust if new types introduced
	case 1, 2, 3:
		return uint(v), true
	default:
		return 0, false
	}
}

func (ws *app) HandleUpdateFields(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	tmpFields := make(map[string]*FieldData)
	err := json.NewDecoder(r.Body).Decode(&tmpFields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for key, field := range tmpFields {
		facet, ok := ws.findFacet(field.Id)
		if ok {
			base := facet.BaseField
			if base != nil {
				if field.Name != "" {
					base.Name = field.Name
				}
				if field.Description != "" {
					base.Description = field.Description
				}
			}
		}
		existing, found := ws.fieldData[key]
		if found {
			if existing.Created == 0 {
				existing.Created = time.Now().UnixMilli()
			}
			existing.Purpose = field.Purpose
			if field.Name != "" {
				existing.Name = field.Name
			}
			if field.Description != "" {
				existing.Description = field.Description
			}
			existing.Type = field.Type
			existing.LastSeen = time.Now().UnixMilli()
		} else {
			field.LastSeen = time.Now().UnixMilli()
			field.Created = time.Now().UnixMilli()
			ws.fieldData[key] = *field
		}
	}
	err = ws.storage.SaveGzippedJson(ws.fieldData, "fields.jz")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = ws.storage.SaveFacets(&ws.storageFacets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *app) GetFields(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(ws.fieldData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *app) GetField(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	fieldId := r.PathValue("id")
	field, ok := ws.fieldData[fieldId]
	if !ok {
		http.Error(w, "Field not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(field)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *app) UpdateFacetsFromFields(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	//toDelete := make([]uint, 0)
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for _, field := range ws.fieldData {
		facet, ok := ws.findFacet(field.Id)
		if ok {
			base := facet.BaseField
			if base != nil {
				base.Name = field.Name
				base.Description = field.Description
				if slices.Index(field.Purpose, "do not show") != -1 {
					base.HideFacet = true
				}
				if slices.Index(field.Purpose, "UL Benchmarking") != -1 {
					base.Type = "fps"
				}
				if slices.Index(field.Purpose, "Key Specification") == -1 {
					base.KeySpecification = false
				} else {
					base.KeySpecification = true
				}
			}
		}
	}
	// for _, id := range toDelete {
	// 	delete(ws.Index.Facets, id)
	// }

	err := ws.storage.SaveFacets(&ws.storageFacets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *app) findFacet(id uint) (*types.StorageFacet, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	// Iterate by index to return pointer to the actual slice element (not loop copy)
	for i := range ws.storageFacets {
		if ws.storageFacets[i].Id == id {
			return &ws.storageFacets[i], true
		}
	}
	return nil, false
}

func (ws *app) CreateFacetFromField(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	fieldId := r.PathValue("id")
	field, ok := ws.fieldData[fieldId]
	if !ok {
		http.Error(w, "Field not found", http.StatusNotFound)
		return
	}
	_, found := ws.findFacet(field.Id)
	if found {
		http.Error(w, "Facet already exists", http.StatusBadRequest)
		return
	}
	baseField := &types.BaseField{
		Name:        field.Name,
		Description: field.Description,
		Id:          field.Id,
		Priority:    10,
		Searchable:  true,
	}
	if slices.Index(field.Purpose, "do not show") != -1 {
		baseField.HideFacet = true
	}
	ft, okType := getFieldType(field.Type)
	if !okType {
		http.Error(w, "Invalid field type", http.StatusBadRequest)
		return
	}
	ws.storageFacets = append(ws.storageFacets, types.StorageFacet{
		BaseField: baseField,
		Type:      types.FieldType(ft),
	})
	err := ws.storage.SaveFacets(&ws.storageFacets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	change := types.FieldChange{
		Action:    types.ADD_FIELD,
		BaseField: baseField,
		FieldType: ft,
	}
	if err = ws.amqpSender.SendFacetChanges(change); err != nil {
		log.Printf("Could not send facet changes: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *app) DeleteFacet(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	facetIdString := r.PathValue("id")
	facetId64, err := strconv.ParseUint(facetIdString, 10, 64)
	if err != nil {
		http.Error(w, "Invalid facet ID", http.StatusBadRequest)
		return
	}
	// Prevent overflow converting to platform uint (gosec G115 mitigation)
	if facetId64 > uint64(^uint(0)) { // additional safety if running on 32-bit
		http.Error(w, "Facet ID out of range", http.StatusBadRequest)
		return
	}
	facetId := uint(facetId64)
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.storageFacets = slices.DeleteFunc(ws.storageFacets, func(f types.StorageFacet) bool {
		return f.Id == facetId
	})

	if err = ws.storage.SaveFacets(&ws.storageFacets); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	change := types.FieldChange{
		Action:    types.REMOVE_FIELD,
		BaseField: &types.BaseField{Id: facetId},
		FieldType: 0,
	}
	if err = ws.amqpSender.SendFacetChanges(change); err != nil {
		log.Printf("Could not send facet changes: %v", err)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *app) UpdateFacet(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	facetIdString := r.PathValue("id")
	facetId64, err := strconv.ParseUint(facetIdString, 10, 64)
	if err != nil {
		http.Error(w, "Invalid facet ID", http.StatusBadRequest)
		return
	}
	if facetId64 > uint64(^uint(0)) {
		http.Error(w, "Facet ID out of range", http.StatusBadRequest)
		return
	}
	facetId := uint(facetId64)
	data := types.BaseField{}
	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	facet, ok := ws.findFacet(facetId)
	if !ok {
		http.Error(w, "Facet not found", http.StatusNotFound)
		return
	}
	current := facet.BaseField
	if current == nil {
		http.Error(w, "Facet not found", http.StatusNotFound)
		return
	}
	current.UpdateFrom(&data)

	if err = ws.storage.SaveFacets(&ws.storageFacets); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	ft, okType := getFieldType(facet.Type)
	if !okType {
		http.Error(w, "Invalid facet type", http.StatusInternalServerError)
		return
	}
	change := types.FieldChange{
		Action:    types.UPDATE_FIELD,
		BaseField: current,
		FieldType: ft,
	}
	if err = ws.amqpSender.SendFacetChanges(change); err != nil {
		log.Printf("Could not send facet changes: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *app) MissingFacets(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	missing := make([]FieldData, 0)
	for _, field := range ws.fieldData {
		_, ok := ws.findFacet(field.Id)
		if !ok {
			missing = append(missing, field)
		}
	}

	err := json.NewEncoder(w).Encode(missing)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
