package cart

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"tornberg.me/facet-search/pkg/index"
)

type CartServer struct {
	Storage   CartStorage
	IdHandler CartIdStorage
	Index     *index.Index
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
	dataItem, ok := s.Index.Items[item.ItemId]
	s.Index.Unlock()
	if !ok {
		return nil, errors.New("item not found")
	}
	cartItem := CartItem{
		Sku:      dataItem.Sku,
		Title:    dataItem.Title,
		Quantity: item.Quantity,
		ImageUrl: dataItem.Img,
	}
	for _, field := range dataItem.IntegerFields {
		if field.Id == 4 {
			cartItem.Price = field.Value
		}
	}

	return &cartItem, nil
}

func (s *CartServer) AddSessionItem(w http.ResponseWriter, req *http.Request) {
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
}

type ChangeQuantity struct {
	Quantity uint `json:"quantity"`
	Id       uint `json:"id"`
}

func (s *CartServer) ChangeQuantitySessionItem(w http.ResponseWriter, req *http.Request) {
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
}

func (s *CartServer) RemoveSessionItem(w http.ResponseWriter, req *http.Request) {
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
