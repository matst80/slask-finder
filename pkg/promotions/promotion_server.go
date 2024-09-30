package promotions

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type PromotionServer struct {
	Storage PromotionStorage
}

func NewPromotionServer(storage PromotionStorage) *PromotionServer {
	return &PromotionServer{
		Storage: storage,
	}
}

func (srv *PromotionServer) GetPromotions(w http.ResponseWriter, req *http.Request) {
	promotions, err := srv.Storage.GetPromotions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error getting promotions"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(promotions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (srv *PromotionServer) AddPromotion(w http.ResponseWriter, req *http.Request) {
	promotion := Promotion{}
	err := json.NewDecoder(req.Body).Decode(&promotion)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid promotion"))
		return
	}
	err = srv.Storage.AddPromotion(promotion)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error adding promotion"))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (srv *PromotionServer) RemovePromotion(w http.ResponseWriter, req *http.Request) {
	idString := req.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid promotion id"))
		return
	}
	err = srv.Storage.RemovePromotion(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error removing promotion"))
		return
	}
}

func (srv *PromotionServer) PromotionHandler() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", srv.GetPromotions)
	mux.HandleFunc("POST /", srv.AddPromotion)
	//mux.HandleFunc("PUT /", srv.ChangeQuantitySessionItem)
	mux.HandleFunc("DELETE /{id}", srv.RemovePromotion)

	return mux
}
