package cart

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type CartInputItem struct {
	ItemId   uint `json:"id"`
	Quantity uint `json:"quantity"`
}

type CartItem struct {
	Sku           string `json:"sku"`
	Title         string `json:"title,omitempty"`
	Price         int    `json:"price"`
	OriginalPrice int    `json:"original_price,omitempty"`
	Quantity      uint   `json:"qty"`
	Id            uint   `json:"id,omitempty"`
	// AddedAt        Timestamp
	// StockValue     int
	// StockUpdatedAt DateTime
	ImageUrl string `json:"image,omitempty"`
}

type Cart struct {
	Id         int        `json:"id"`
	Items      []CartItem `json:"items"`
	TotalPrice int        `json:"total_price"`
}

type CartStorage interface {
	AddItem(cartId int, item *CartItem) (*Cart, error)
	ChangeQuantity(cartId int, id uint, quantity uint) (*Cart, error)
	RemoveItem(cartId int, id uint) (*Cart, error)
	GetCart(cartId int) (*Cart, error)
}

type CartIdStorage interface {
	GetNextCartId() (int, error)
}

type DiskCartStorage struct {
	Path string
}

func NewDiskCartStorage(path string) *DiskCartStorage {
	return &DiskCartStorage{
		Path: path,
	}
}

func (s *DiskCartStorage) readFile(path string, dest any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(dest)
	return err
}

func (s *DiskCartStorage) writeFile(path string, src any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	err = json.NewEncoder(file).Encode(src)
	return err
}

func (s *DiskCartStorage) GetNextCartId() (int, error) {
	id := 0
	err := s.readFile(filepath.Join(s.Path, "next_id"), &id)
	if err != nil {
		return 0, err
	}
	err = s.writeFile(filepath.Join(s.Path, "next_id"), id+1)
	if err != nil {
		return 0, err
	}
	return id + 1, nil
}

func getFolder(id int) (string, string) {
	return fmt.Sprintf("%d", id/1000), fmt.Sprintf("%d.json", id%1000)
}

func cartCartPrice(cart *Cart) int {
	total := 0
	for _, item := range cart.Items {
		total += item.Price * int(item.Quantity)
	}
	return total
}

func (s *DiskCartStorage) ChangeQuantity(cartId int, id uint, quantity uint) (*Cart, error) {
	cart, err := s.GetCart(cartId)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, fmt.Errorf("cart %d not found", cartId)
	}
	for i, item := range cart.Items {
		if item.Id == id {
			cart.Items[i].Quantity = quantity
			cart.TotalPrice = cartCartPrice(cart)
			err = s.saveCart(cart)
			if err != nil {
				return nil, err
			}
			return cart, nil
		}
	}
	return nil, fmt.Errorf("item %d not found in cart %d", id, cartId)
}

func (s *DiskCartStorage) RemoveItem(cartId int, id uint) (*Cart, error) {
	cart, err := s.GetCart(cartId)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, fmt.Errorf("cart %d not found", cartId)
	}
	for i, item := range cart.Items {
		if item.Id == id {
			cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			cart.TotalPrice = cartCartPrice(cart)
			err = s.saveCart(cart)
			if err != nil {
				return nil, err
			}
			return cart, nil
		}
	}
	return nil, fmt.Errorf("item %d not found in cart %d", id, cartId)
}

func (s *DiskCartStorage) AddItem(cartId int, item *CartItem) (*Cart, error) {
	cart, err := s.GetCart(cartId)
	if err != nil {
		log.Printf("Creating new cart %d", cartId)
	}
	if cart == nil {
		cart = &Cart{
			Id: cartId,
		}
	}
	cart.Items = append(cart.Items, *item)
	for idx, _ := range cart.Items {
		cart.Items[idx].Id = uint(idx)
	}
	cart.TotalPrice = cartCartPrice(cart)
	err = s.saveCart(cart)
	if err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *DiskCartStorage) GetCart(cartId int) (*Cart, error) {
	folder, filename := getFolder(cartId)
	path := filepath.Join(s.Path, folder, filename)

	var cart Cart
	err := s.readFile(path, &cart)
	if err != nil {
		return nil, err
	}
	return &cart, nil
}

func (s *DiskCartStorage) saveCart(cart *Cart) error {
	folder, filename := getFolder(cart.Id)
	if err := os.MkdirAll(filepath.Join(s.Path, folder), 0755); err != nil {
		return err
	}
	path := filepath.Join(s.Path, folder, filename)
	return s.writeFile(path, cart)
}
