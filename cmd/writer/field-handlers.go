package main

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/types"
)

func (ws *MasterApp) HandleUpdateFields(w http.ResponseWriter, r *http.Request) {
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
	err = ws.storage.SaveFacets(ws.storageFacets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *MasterApp) GetFields(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(ws.fieldData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *MasterApp) GetField(w http.ResponseWriter, r *http.Request) {
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

func (ws *MasterApp) UpdateFacetsFromFields(w http.ResponseWriter, r *http.Request) {
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

	err := ws.storage.SaveFacets(ws.storageFacets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *MasterApp) findFacet(id uint) (*facet.StorageFacet, bool) {
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

func (ws *MasterApp) CreateFacetFromField(w http.ResponseWriter, r *http.Request) {
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
	ws.storageFacets = append(ws.storageFacets, facet.StorageFacet{
		BaseField: baseField,
		Type:      facet.FieldType(field.Type),
	})
	err := ws.storage.SaveFacets(ws.storageFacets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	change := types.FieldChange{
		Action:    types.ADD_FIELD,
		BaseField: baseField,
		FieldType: uint(field.Type),
	}
	if err = ws.amqpSender.SendFacetChanges(change); err != nil {
		log.Printf("Could not send facet changes: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *MasterApp) DeleteFacet(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	facetIdString := r.PathValue("id")
	facetId, err := strconv.Atoi(facetIdString)
	if err != nil {
		http.Error(w, "Invalid facet ID", http.StatusBadRequest)
		return
	}
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.storageFacets = slices.DeleteFunc(ws.storageFacets, func(f facet.StorageFacet) bool {
		return f.Id == uint(facetId)
	})

	if err = ws.storage.SaveFacets(ws.storageFacets); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	change := types.FieldChange{
		Action:    types.REMOVE_FIELD,
		BaseField: &types.BaseField{Id: uint(facetId)},
		FieldType: 0,
	}
	if err = ws.amqpSender.SendFacetChanges(change); err != nil {
		log.Printf("Could not send facet changes: %v", err)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *MasterApp) UpdateFacet(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	facetIdString := r.PathValue("id")
	facetId, err := strconv.Atoi(facetIdString)
	if err != nil {
		http.Error(w, "Invalid facet ID", http.StatusBadRequest)
		return
	}
	data := types.BaseField{}
	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	facet, ok := ws.findFacet(uint(facetId))
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

	if err = ws.storage.SaveFacets(ws.storageFacets); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	change := types.FieldChange{
		Action:    types.UPDATE_FIELD,
		BaseField: current,
		FieldType: uint(facet.Type),
	}
	if err = ws.amqpSender.SendFacetChanges(change); err != nil {
		log.Printf("Could not send facet changes: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (ws *MasterApp) MissingFacets(w http.ResponseWriter, r *http.Request) {
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
