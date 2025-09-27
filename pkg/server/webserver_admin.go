package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalItems = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "slaskfinder_items_total",
		Help: "The total number of items in index",
	})
)

func (ws *AdminWebServer) HandlePopularRules(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		defaultHeaders(w, r, false, "0")
		jsonArray := types.JsonTypes{}
		err := json.NewDecoder(r.Body).Decode(&jsonArray)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sort := make(types.ItemPopularityRules, 0, len(jsonArray))
		for _, item := range jsonArray {
			v, ok := item.(types.ItemPopularityRule)
			if !ok {
				continue
			}
			sort = append(sort, v)
		}
		types.CurrentSettings.PopularityRules = &sort
		//ws.Sorting.SetPopularityRules(&sort)

		w.WriteHeader(http.StatusOK)
		return
	}

	sort := types.CurrentSettings.PopularityRules
	if sort == nil {
		http.Error(w, "rules not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)

	jsonArray := make(types.JsonTypes, 0, len(*sort))
	for _, v := range *sort {
		j, ok := v.(types.JsonType)
		if !ok {
			continue
		}
		jsonArray = append(jsonArray, j)
	}

	err := json.NewEncoder(w).Encode(jsonArray)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) HandlePopularOverride(w http.ResponseWriter, r *http.Request) {
	if ws.Sorting == nil {
		log.Printf("Sorting not initialized in handlePopularOverride")
		http.Error(w, "Sorting not initialized", http.StatusInternalServerError)
		return
	}
	if r.Method == "POST" {
		defaultHeaders(w, r, true, "0")
		sort := index.SortOverride{}
		err := json.NewDecoder(r.Body).Decode(&sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Sorting.AddPopularOverride(&sort)

		w.WriteHeader(http.StatusOK)
		return
	}

	sort := ws.Sorting.GetPopularOverrides()
	if sort == nil {
		http.Error(w, "Sort not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) AddItem(w http.ResponseWriter, r *http.Request) {
	items := AddItemRequest{}
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	toUpdate := make([]types.Item, len(items))
	for i, item := range items {
		// Send notifications for price watchers
		if ws.PriceWatches != nil {
			ws.PriceWatches.NotifyPriceWatchers(&item)
		}
		toUpdate[i] = &item
	}
	ws.Index.HandleItems(toUpdate)
	totalItems.Set(float64(len(ws.Index.Items)))
	toUpdate = nil
	w.WriteHeader(http.StatusOK)
}

func (ws *AdminWebServer) Save(w http.ResponseWriter, _ *http.Request) {
	err := ws.Db.SaveIndex(ws.Index)
	if err != nil {
		log.Printf("Error saving index: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// type CategoryUpdateRequest struct {
// 	Ids     []uint                 `json:"ids"`
// 	Updates []types.CategoryUpdate `json:"updates"`
// }

// func (ws *WebServer) UpdateCategories(w http.ResponseWriter, r *http.Request) {
// 	update := CategoryUpdateRequest{}
// 	err := json.NewDecoder(r.Body).Decode(&update)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	ws.Index.UpdateCategoryValues(update.Ids, update.Updates)
// 	w.WriteHeader(http.StatusAccepted)
// }

//func (ws *WebServer) RagData(w http.ResponseWriter, r *http.Request) {
//	defaultHeaders(w, false, "240")
//	w.WriteHeader(http.StatusOK)
//	ws.Index.Lock()
//	defer ws.Index.Unlock()
//	sorting := ws.Index.Sorting.GetSort("popular")
//
//	for i, id := range *sorting {
//		item, ok := ws.Index.Items[id]
//		if !ok {
//			continue
//		}
//		fmt.Fprintf(w, (*item).ToString())
//
//		w.Write([]byte("\n"))
//		if i > 500 {
//			break
//		}
//	}
//
//}

func (ws *AdminWebServer) HandleUpdateFields(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	tmpFields := make(map[string]*FieldData)
	err := json.NewDecoder(r.Body).Decode(&tmpFields)
	for key, field := range tmpFields {
		facet, ok := ws.FacetHandler.Facets[field.Id]
		if ok {
			base := facet.GetBaseField()
			if base != nil {
				if field.Name != "" {
					base.Name = field.Name
				}
				if field.Description != "" {
					base.Description = field.Description
				}
			}
		}
		existing, found := ws.FieldData[key]
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
			ws.FieldData[key] = field
		}
	}
	ws.Db.SaveJsonFile(ws.FieldData, "fields.jz")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) GetFields(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(ws.FieldData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) GetField(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	fieldId := r.PathValue("id")
	field, ok := ws.FieldData[fieldId]
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

func (ws *AdminWebServer) CleanFields(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	ws.Index.Lock()
	defer ws.Index.Unlock()
	for _, field := range ws.FieldData {
		field.ItemCount = 0
	}
	for _, itemP := range ws.Index.Items {
		item, ok := itemP.(*index.DataItem)
		if ok && !item.IsDeleted() {
			for _, field := range ws.FieldData {
				_, found := item.Fields[field.Id]
				if found {
					field.ItemCount++
				}
			}
		}
	}
	cleanFields := make(map[string]*FieldData)
	for key, field := range ws.FieldData {
		if field.ItemCount > 0 {
			cleanFields[key] = field
			facet, ok := ws.FacetHandler.Facets[field.Id]
			if ok {
				base := facet.GetBaseField()
				if base != nil {
					base.Name = field.Name
					base.Description = field.Description
					if slices.Index(field.Purpose, "Key Specification") == -1 {
						base.KeySpecification = false
					} else {
						base.KeySpecification = true
					}
				}
			} else {
				log.Printf("Field %s not found in index", key)
			}
		}
	}
	ws.FieldData = cleanFields
	err := ws.Db.SaveJsonFile(ws.FieldData, "fields.jz")
	// err := json.NewEncoder(w).Encode(cleanFields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *AdminWebServer) UpdateFacetsFromFields(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	//toDelete := make([]uint, 0)
	for _, field := range ws.FieldData {
		facet, ok := ws.FacetHandler.Facets[field.Id]
		if ok {
			base := facet.GetBaseField()
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
			if field.ItemCount < 5 {
				log.Printf("Useless index field %s %d, count: %d", field.Name, field.Id, field.ItemCount)
				// 	toDelete = append(toDelete, field.Id)
			}
		}
	}
	// for _, id := range toDelete {
	// 	delete(ws.Index.Facets, id)
	// }

	err := ws.Db.SaveFacets(ws.FacetHandler.Facets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) CreateFacetFromField(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	fieldId := r.PathValue("id")
	field, ok := ws.FieldData[fieldId]
	if !ok {
		http.Error(w, "Field not found", http.StatusNotFound)
		return
	}
	_, found := ws.FacetHandler.Facets[field.Id]
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
	switch field.Type {
	case KEY:
		ws.FacetHandler.AddKeyField(baseField)
	case NUMBER:
		ws.FacetHandler.AddIntegerField(baseField)
	case DECIMAL:
		ws.FacetHandler.AddDecimalField(baseField)
	}
	err := ws.Db.SaveFacets(ws.FacetHandler.Facets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *AdminWebServer) DeleteFacet(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	facetIdString := r.PathValue("id")
	facetId, err := strconv.Atoi(facetIdString)
	if err != nil {
		http.Error(w, "Invalid facet ID", http.StatusBadRequest)
		return
	}
	_, ok := ws.FacetHandler.Facets[uint(facetId)]
	if !ok {
		http.Error(w, "Facet not found", http.StatusNotFound)
		return
	}
	delete(ws.FacetHandler.Facets, uint(facetId))
	if err = ws.Db.SaveFacets(ws.FacetHandler.Facets); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *AdminWebServer) UpdateFacet(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
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

	facet, ok := ws.FacetHandler.Facets[uint(facetId)]
	if !ok {
		http.Error(w, "Facet not found", http.StatusNotFound)
		return
	}
	current := facet.GetBaseField()
	if current == nil {
		http.Error(w, "Facet not found", http.StatusNotFound)
		return
	}
	current.UpdateFrom(&data)

	if err = ws.Db.SaveFacets(ws.FacetHandler.Facets); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	changes := make([]types.FieldChange, 0)
	changes = append(changes, types.FieldChange{
		Action:    types.UPDATE_FIELD,
		BaseField: current,
		FieldType: facet.GetType(),
	})
	if ws.FacetHandler.ChangeHandler != nil {
		ws.FacetHandler.ChangeHandler.FieldsChanged(changes)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *AdminWebServer) MissingFacets(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	missing := make([]*FieldData, 0)
	for _, field := range ws.FieldData {
		_, ok := ws.FacetHandler.Facets[field.Id]
		if !ok {
			missing = append(missing, field)
		}
	}

	err := json.NewEncoder(w).Encode(missing)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) GetSettings(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	if r.Method == http.MethodPut {
		types.CurrentSettings.Lock()
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings)
		types.CurrentSettings.Unlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = ws.Db.SaveSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) GetSearchIndexedFacets(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "5")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings.FieldsToIndex)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) SetSearchIndexedFacets(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	if r.Method == "POST" {
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings.FieldsToIndex)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Db.SaveSettings()
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *AdminWebServer) HandleFacetGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		types.CurrentSettings.Lock()
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings.FacetGroups)
		types.CurrentSettings.Unlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = ws.Db.SaveSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	defaultHeaders(w, r, true, "1200")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings.FacetGroups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *AdminWebServer) GetFacetList(w http.ResponseWriter, r *http.Request) {
	publicHeaders(w, r, true, "10")

	w.WriteHeader(http.StatusOK)

	res := make([]types.BaseField, len(ws.FacetHandler.Facets))
	idx := 0
	for _, f := range ws.FacetHandler.Facets {
		res[idx] = *f.GetBaseField()
		idx++
	}
	enc := json.NewEncoder(w)

	enc.Encode(res)
}

type FacetGroupingData struct {
	GroupId   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
	FacetIds  []uint `json:"facet_ids"`
}

func (ws *AdminWebServer) FacetGroupUpdate(w http.ResponseWriter, r *http.Request) {
	data := FacetGroupingData{}
	json.NewDecoder(r.Body).Decode(&data)
	if data.GroupId == 0 {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}
	changes := make([]types.FieldChange, 0)

	types.CurrentSettings.Lock()
	defer types.CurrentSettings.Unlock()
	idx := slices.IndexFunc(types.CurrentSettings.FacetGroups, func(g types.FacetGroup) bool {
		return g.Id == data.GroupId
	})
	if idx == -1 {
		if len(data.GroupName) == 0 {
			http.Error(w, "Group name is required", http.StatusBadRequest)
			return
		}
		types.CurrentSettings.FacetGroups = append(types.CurrentSettings.FacetGroups, types.FacetGroup{
			Id:   data.GroupId,
			Name: data.GroupName,
		})
	}

	for _, id := range data.FacetIds {
		facet, ok := ws.FacetHandler.Facets[id]
		if !ok {
			continue
		}
		base := facet.GetBaseField()
		if base != nil && base.GroupId != data.GroupId {
			base.GroupId = data.GroupId

			changes = append(changes, types.FieldChange{
				Action:    types.UPDATE_FIELD,
				BaseField: base,
				FieldType: facet.GetType(),
			})
		}
	}

	if ws.FacetHandler.ChangeHandler != nil {
		ws.FacetHandler.ChangeHandler.FieldsChanged(changes)
	}
	err := ws.Db.SaveFacets(ws.FacetHandler.Facets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type MatchedRule struct {
	Rule  interface{} `json:"rule"`
	Score float64     `json:"score"`
}

type PopularityResult struct {
	Popularity float64       `json:"popularity"`
	Matches    []MatchedRule `json:"matches"`
}

func (ws *AdminWebServer) GetItemPopularity(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "0")
	itemIdString := r.PathValue("id")
	itemId, err := strconv.Atoi(itemIdString)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}
	item, ok := ws.Index.Items[uint(itemId)]
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	ret := &PopularityResult{
		Popularity: 0,
		Matches:    make([]MatchedRule, 0),
	}
	rules := types.CurrentSettings.PopularityRules
	for _, rule := range *rules {
		if rule == nil {
			continue
		}
		score := rule.GetValue(item)
		if score != 0 {
			ret.Popularity += score
			ret.Matches = append(ret.Matches, MatchedRule{
				Rule:  rule,
				Score: score,
			})
		}
	}

	err = json.NewEncoder(w).Encode(ret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type WordReplacementConfig struct {
	SplitWords   []string          `json:"splitWords"`
	WordMappings map[string]string `json:"wordMappings"`
}

func (ws *AdminWebServer) SaveEmbeddings(w http.ResponseWriter, r *http.Request) {

	if err := ws.Db.SaveEmbeddings(ws.EmbeddingsHandler.GetAllEmbeddings()); err != nil {
		log.Printf("Error saving embeddings: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (ws *AdminWebServer) HandleWordReplacements(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		data := WordReplacementConfig{}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		types.CurrentSettings.Lock()
		defer types.CurrentSettings.Unlock()
		types.CurrentSettings.WordMappings = data.WordMappings
		types.CurrentSettings.SplitWords = data.SplitWords
	}
	ret := WordReplacementConfig{
		WordMappings: types.CurrentSettings.WordMappings,
		SplitWords:   types.CurrentSettings.SplitWords,
	}
	err := json.NewEncoder(w).Encode(ret)
	if err != nil {
		log.Printf("unable to respond: %v", err)
	}
}

func (ws *AdminWebServer) GetItem(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	id := r.PathValue("id")
	itemId, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	item, ok := ws.Index.Items[uint(itemId)]
	if !ok {
		return fmt.Errorf("item not found")
	}
	publicHeaders(w, r, true, "10")
	w.WriteHeader(http.StatusOK)
	return enc.Encode(item)
}

func (ws *AdminWebServer) Handle() *http.ServeMux {
	config := &webauthn.Config{
		RPDisplayName: "Go WebAuthn",
		RPID:          "slask-finder.tornberg.me",
		RPOrigins:     []string{"https://slask-finder.tornberg.me", "https://slask-finder.knatofs.se"},
	}

	priceWatcher := NewPriceWatcher()
	ws.PriceWatches = priceWatcher

	auth, err := NewWebAuthHandler(config)
	if err != nil {
		log.Fatalf("Error initializing WebAuthn: %v", err)
	}

	srv := http.NewServeMux()
	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		defaultHeaders(w, r, false, "0")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			log.Println("Error writing health check response")
		}
	})
	tmp := map[string]*FieldData{}
	ws.Db.LoadJsonFile(&tmp, "fields.jz")
	for key, field := range tmp {
		if field.Name == "Supplier's EcoVadis Score" {
			continue
		}
		if field.Name == "Audio control" {
			continue
		}
		if field.Name == "Suitable for commercial use" {
			continue
		}
		if field.Name == "Suitable for children" {
			continue
		}
		ws.FieldData[key] = field
	}
	tmp = nil

	srv.HandleFunc("/login", ws.Login)
	srv.HandleFunc("/logout", ws.Logout)
	srv.HandleFunc("/user", ws.User)
	//srv.HandleFunc("GET /rag", ws.RagData)
	srv.HandleFunc("/auth_callback", ws.AuthCallback)
	srv.HandleFunc("/add", ws.AuthMiddleware(ws.AddItem))

	//srv.HandleFunc("PUT /key-values", ws.AuthMiddleware(ws.UpdateCategories))
	srv.HandleFunc("/save", ws.AuthMiddleware(ws.Save))
	srv.HandleFunc("/store-embeddings", ws.AuthMiddleware(ws.SaveEmbeddings))
	srv.HandleFunc("PUT /fields", ws.AuthMiddleware(ws.HandleUpdateFields))
	srv.HandleFunc("/clean-fields", ws.CleanFields)
	srv.HandleFunc("/update-fields", ws.UpdateFacetsFromFields)
	srv.HandleFunc("DELETE /facets/{id}", ws.AuthMiddleware(ws.DeleteFacet))
	srv.HandleFunc("GET /facets", ws.GetFacetList)
	srv.HandleFunc("PUT /facets/{id}", ws.AuthMiddleware(ws.UpdateFacet))
	srv.HandleFunc("GET /index/facets", ws.AuthMiddleware(ws.GetSearchIndexedFacets))
	srv.HandleFunc("POST /index/facets", ws.AuthMiddleware(ws.SetSearchIndexedFacets))
	srv.HandleFunc("GET /item/{id}/popularity", ws.AuthMiddleware(ws.GetItemPopularity))
	srv.HandleFunc("GET /fields/{id}/add", ws.AuthMiddleware(ws.CreateFacetFromField))
	srv.HandleFunc("GET /fields", ws.GetFields)
	srv.HandleFunc("GET /item/{id}", ws.AuthMiddleware(JsonHandler(ws.Tracking, ws.GetItem)))
	srv.HandleFunc("GET /settings", ws.GetSettings)
	srv.HandleFunc("PUT /settings", ws.AuthMiddleware(ws.UpdateSettings))
	srv.HandleFunc("PUT /facet-group", ws.AuthMiddleware(ws.FacetGroupUpdate))
	srv.HandleFunc("/words", ws.AuthMiddleware(ws.HandleWordReplacements))

	srv.HandleFunc("POST /price-watch/{id}", priceWatcher.WatchPriceChange)

	srv.HandleFunc("GET /missing-fields", ws.AuthMiddleware(ws.MissingFacets))
	srv.HandleFunc("GET /fields/{id}", ws.GetField)
	srv.HandleFunc("/rules/popular", ws.AuthMiddleware(ws.HandlePopularRules))
	srv.HandleFunc("/sort/popular", ws.AuthMiddleware(ws.HandlePopularOverride))
	srv.HandleFunc("POST /relation-groups", ws.SaveHandleRelationGroups)
	srv.HandleFunc("/facet-groups", ws.HandleFacetGroups)

	srv.HandleFunc("GET /users", ws.AuthMiddleware(auth.ListUsers))
	srv.HandleFunc("DELETE /users/{id}", ws.AuthMiddleware(auth.DeleteUser))
	srv.HandleFunc("PUT /users/{id}", ws.AuthMiddleware(auth.UpdateUser))

	srv.HandleFunc("GET /webauthn/register/start", auth.CreateChallenge)
	srv.HandleFunc("POST /webauthn/register/finish", auth.ValidateCreateChallengeResponse)
	srv.HandleFunc("GET /webauthn/login/start", auth.LoginChallenge)
	srv.HandleFunc("POST /webauthn/login/finish", auth.LoginChallengeResponse)
	//srv.HandleFunc("/sort/static", ws.AuthMiddleware(ws.HandleStaticPositions))
	//srv.HandleFunc("/sort/fields", ws.AuthMiddleware(ws.HandleFieldSort))
	return srv
}
