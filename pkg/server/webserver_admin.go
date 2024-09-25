package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tornberg.me/facet-search/pkg/facet"
)

func (ws *WebServer) HandleSortId(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if r.Method == "POST" {
		defaultHeaders(w, true, "0")
		sort := facet.SortIndex{}
		err := json.NewDecoder(r.Body).Decode(&sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = ws.Sorting.AddSortMethod(id, &sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	sort := ws.Sorting.GetSort(id)
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
		sort := facet.SortIndex{}
		err := json.NewDecoder(r.Body).Decode(&sort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws.Sorting.SetFieldSort(&sort)
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

func (ws *WebServer) AddItem(w http.ResponseWriter, r *http.Request) {
	items := AddItemRequest{}
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, item := range items {
		ws.Index.UpsertItem(&item)
	}
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
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
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
	srv.HandleFunc("/save", ws.Save)
	srv.HandleFunc("/sort/{id}", ws.HandleSortId)
	srv.HandleFunc("/field-sort", ws.HandleFieldSort)
	return srv
}
