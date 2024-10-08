package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tornberg.me/facet-search/pkg/index"
)

func (ws *WebServer) HandlePopularOverride(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		defaultHeaders(w, true, "0")
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
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) HandleFieldSort(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		defaultHeaders(w, true, "0")
		sort := index.SortOverride{}
		err := json.NewDecoder(r.Body).Decode(&sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Sorting.SetFieldSortOverride(&sort)
		w.WriteHeader(http.StatusOK)
		return
	}

	sort := ws.Sorting.FieldSort
	if sort == nil {
		http.Error(w, "Sort not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) HandleStaticPositions(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		defaultHeaders(w, true, "0")
		sort := index.StaticPositions{}
		err := json.NewDecoder(r.Body).Decode(&sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Sorting.SetStaticPositions(sort)
		w.WriteHeader(http.StatusOK)
		return
	}

	sort := ws.Sorting.GetStaticPositions()
	if sort == nil {
		http.Error(w, "Sort not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(sort)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) AddItem(w http.ResponseWriter, r *http.Request) {
	items := AddItemRequest{}
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws.Index.UpsertItems(items)

	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) GetItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	itemId, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item, ok := ws.Index.Items[uint(itemId)]
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) Save(w http.ResponseWriter, r *http.Request) {
	err := ws.Db.SaveIndex(ws.Index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type CategoryUpdateRequest struct {
	Ids     []uint                 `json:"ids"`
	Updates []index.CategoryUpdate `json:"updates"`
}

func (ws *WebServer) UpdateCategories(w http.ResponseWriter, r *http.Request) {
	update := CategoryUpdateRequest{}
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws.Index.UpdateCategoryValues(update.Ids, update.Updates)
	w.WriteHeader(http.StatusAccepted)
}

func (ws *WebServer) AdminHandler() *http.ServeMux {
	srv := http.NewServeMux()
	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		defaultHeaders(w, false, "0")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	srv.HandleFunc("/add", ws.AddItem)
	srv.HandleFunc("/get/{id}", ws.GetItem)
	srv.HandleFunc("PUT /key-values", ws.UpdateCategories)
	srv.HandleFunc("/save", ws.Save)
	srv.HandleFunc("/sort/popular", ws.HandlePopularOverride)
	srv.HandleFunc("/sort/static", ws.HandleStaticPositions)
	//srv.HandleFunc("/sort/{id}/partial", ws.ReOrderSort)
	srv.HandleFunc("/sort/fields", ws.HandleFieldSort)
	return srv
}
