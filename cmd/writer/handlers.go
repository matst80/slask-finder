package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/matst80/slask-finder/pkg/types"
)

func (app *app) dummyResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ws *app) GetSettings(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *app) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	if r.Method == http.MethodPut {
		types.CurrentSettings.Lock()
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings)
		types.CurrentSettings.Unlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = ws.storage.SaveSettings()

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

func (ws *app) GetFacetList(w http.ResponseWriter, r *http.Request) {
	//publicHeaders(w, r, true, "10")

	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)

	err := enc.Encode(ws.storageFacets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *app) HandleWordReplacements(w http.ResponseWriter, r *http.Request) {
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
	ws.storage.SaveSettings()

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(ret)
	if err != nil {
		log.Printf("unable to respond: %v", err)
	}
}

func (ws *app) HandlePopularRules(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		//defaultHeaders(w, r, false, "0")

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

	rules := types.CurrentSettings.PopularityRules
	if rules == nil {
		http.Error(w, "rules not found", http.StatusNotFound)
		return
	}
	//defaultHeaders(w, r, true, "0")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	jsonArray := make(types.JsonTypes, 0, len(*rules))
	for _, v := range *rules {
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

func (ws *app) SaveHandleRelationGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	types.CurrentSettings.Lock()
	err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings.FacetRelations)
	types.CurrentSettings.Unlock()

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = ws.storage.SaveSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (ws *app) HandleFacetGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		types.CurrentSettings.Lock()
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings.FacetGroups)
		types.CurrentSettings.Unlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = ws.storage.SaveSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	//defaultHeaders(w, r, true, "1200")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings.FacetGroups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
