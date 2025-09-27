package main

import (
	"encoding/json"
	"net/http"

	"github.com/matst80/slask-finder/pkg/types"
)

func (app *MasterApp) dummyResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ws *MasterApp) GetSettings(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *MasterApp) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	//defaultHeaders(w, r, true, "0")
	if r.Method == http.MethodPut {
		types.CurrentSettings.Lock()
		err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings)
		types.CurrentSettings.Unlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = ws.storage.SaveJson(types.CurrentSettings, "settings.json")

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

func (ws *MasterApp) GetFacetList(w http.ResponseWriter, r *http.Request) {
	//publicHeaders(w, r, true, "10")

	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)

	err := enc.Encode(ws.storageFacets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
