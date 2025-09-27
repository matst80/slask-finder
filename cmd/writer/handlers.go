package main

import (
	"encoding/json"
	"iter"
	"net/http"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
)

func asSeq(items []index.DataItem) iter.Seq[types.Item] {
	return func(yield func(types.Item) bool) {
		for i := range items {
			if !yield(&items[i]) { // use index to avoid pointer to loop copy
				return
			}
		}
	}
}

func (app *MasterApp) saveItems(w http.ResponseWriter, r *http.Request) {
	// err := app.storage.SaveItems(app.itemIndex.GetAllItems())
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	w.WriteHeader(http.StatusOK)
}

func (app *MasterApp) handleItems(w http.ResponseWriter, r *http.Request) {
	// items := make([]index.DataItem, 0)
	// err := json.NewDecoder(r.Body).Decode(&items)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	// app.itemIndex.HandleItems(asSeq(items))

	// app.amqpSender.SendItems(items)

	w.WriteHeader(http.StatusOK)
}

// func (app *MasterApp) getAdminItemById(w http.ResponseWriter, r *http.Request) {
// 	idValue := r.PathValue("id")
// 	id, err := strconv.ParseUint(idValue, 10, 64)
// 	if err != nil {
// 		http.Error(w, "invalid id", http.StatusBadRequest)
// 		return
// 	}

// 	item, ok := app.itemIndex.GetItem(uint(id))
// 	if !ok {
// 		http.Error(w, "item not found", http.StatusNotFound)
// 		return
// 	}
// 	w.WriteHeader(http.StatusOK)
// 	w.Header().Set("Content-Type", "application/json")
// 	if err := json.NewEncoder(w).Encode(item); err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

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

	enc.Encode(ws.storageFacets)
}
