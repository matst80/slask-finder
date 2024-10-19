package cart

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"tornberg.me/facet-search/pkg/common"
	"tornberg.me/facet-search/pkg/index"

	"tornberg.me/facet-search/pkg/promotions"
	"tornberg.me/facet-search/pkg/tracking"
)

type CartServer struct {
	Storage   CartStorage
	IdHandler CartIdStorage
	Index     *index.Index
	Tracking  tracking.Tracking
}

func (s *CartServer) AddItem(w http.ResponseWriter, req *http.Request) {
	idString := req.PathValue("id")
	cartId, err := strconv.Atoi(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid cart id"))
		return
	}
	var item CartItem
	err = json.NewDecoder(req.Body).Decode(&item)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid item"))
		return
	}
	cart, err := s.Storage.AddItem(cartId, &item)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error adding item"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *CartServer) GetCart(w http.ResponseWriter, req *http.Request) {
	idString := req.PathValue("id")
	cartId, err := strconv.Atoi(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid cart id"))
		return
	}
	cart, err := s.Storage.GetCart(cartId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error getting cart"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cart)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *CartServer) GetSessionCart(w http.ResponseWriter, req *http.Request) {
	cartId, err := handleCartCookie(nil, w, req)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("No cart session"))
		return
	}

	cart, err := s.Storage.GetCart(cartId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error getting cart"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *CartServer) GetCartItem(item *CartInputItem) (*CartItem, error) {
	s.Index.Lock()
	idxItem, ok := s.Index.Items[item.ItemId]
	s.Index.Unlock()
	if !ok {
		return nil, errors.New("item not found")
	}
	dataItem := idxItem.GetBaseItem()
	cartItem := CartItem{
		PromotionInput: &promotions.PromotionInput{},
		Title:          dataItem.Title,
		ImageUrl:       dataItem.Img,
		Id:             item.ItemId,
	}
	cartItem.Sku = dataItem.Sku
	cartItem.Quantity = item.Quantity
	cartItem.Price = dataItem.Price

	return &cartItem, nil
}

func (s *CartServer) AddSessionItem(w http.ResponseWriter, req *http.Request) {
	session_id := common.HandleSessionCookie(s.Tracking, w, req)
	cartId, err := handleCartCookie(s.IdHandler, w, req)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Unable to create cart session"))
		return
	}
	var item CartInputItem
	err = json.NewDecoder(req.Body).Decode(&item)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid item"))
		return
	}
	dataItem, err := s.GetCartItem(&item)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	cart, err := s.Storage.AddItem(cartId, dataItem)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error adding item"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if s.Tracking != nil {
		s.Tracking.TrackAddToCart(uint32(session_id), item.ItemId, item.Quantity)
	}
}

type ChangeQuantity struct {
	Quantity uint `json:"quantity"`
	Id       uint `json:"id"`
}

func (s *CartServer) ChangeQuantitySessionItem(w http.ResponseWriter, req *http.Request) {
	session_id := common.HandleSessionCookie(s.Tracking, w, req)
	cartId, err := handleCartCookie(nil, w, req)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Unable to create cart session"))
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid item id"))
		return
	}
	var item ChangeQuantity
	err = json.NewDecoder(req.Body).Decode(&item)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid item"))
		return
	}
	cart := &Cart{}

	if item.Quantity == 0 {
		cart, err = s.Storage.RemoveItem(cartId, item.Id)
	} else {
		cart, err = s.Storage.ChangeQuantity(cartId, item.Id, item.Quantity)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error changing quantity"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if s.Tracking != nil {
		s.Tracking.TrackAddToCart(uint32(session_id), item.Id, item.Quantity)
	}
}

func (s *CartServer) RemoveSessionItem(w http.ResponseWriter, req *http.Request) {
	session_id := common.HandleSessionCookie(s.Tracking, w, req)
	cartId, err := handleCartCookie(nil, w, req)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Unable to create cart session"))
		return
	}
	idString := req.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid item id"))
		return
	}
	cart, err := s.Storage.RemoveItem(cartId, uint(id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error removing item"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if s.Tracking != nil {
		s.Tracking.TrackAddToCart(uint32(session_id), uint(id), 0)
	}

}

func (srv *CartServer) CartHandler() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", srv.GetSessionCart)
	mux.HandleFunc("POST /", srv.AddSessionItem)
	mux.HandleFunc("PUT /", srv.ChangeQuantitySessionItem)
	mux.HandleFunc("DELETE /{id}", srv.RemoveSessionItem)
	mux.HandleFunc("POST /{id}", srv.AddItem)
	mux.HandleFunc("GET /{id}", srv.GetCart)
	return mux
}
