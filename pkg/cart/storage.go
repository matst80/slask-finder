package cart

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"tornberg.me/facet-search/pkg/promotions"
)

type CartInputItem struct {
	ItemId   uint `json:"id"`
	Quantity uint `json:"quantity"`
}

type CartItem struct {
	*promotions.PromotionInput
	Title         string `json:"title,omitempty"`
	OriginalPrice int    `json:"original_price,omitempty"`
	Id            uint   `json:"id,omitempty"`
	TaxAmount     int    `json:"tax,omitempty"`
	DeliveryId    int    `json:"shipping_option,omitempty"`
	// AddedAt        Timestamp
	// StockValue     int
	// StockUpdatedAt DateTime
	ImageUrl string `json:"image,omitempty"`
}

type Delivery struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	Type        string `json:"type"`
}

type Cart struct {
	Id                int                    `json:"id"`
	Items             []CartItem             `json:"items"`
	AppliedPromotions []promotions.Promotion `json:"applied_promotions"`
	TotalPrice        int                    `json:"total_price"`
	TaxAmount         int                    `json:"tax"`
	Deliveries        []Delivery             `json:"deliveries"`
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
	Path             string
	PromotionStorage promotions.PromotionStorage
}

func NewDiskCartStorage(path string, promotionStorage promotions.PromotionStorage) *DiskCartStorage {
	return &DiskCartStorage{
		Path:             path,
		PromotionStorage: promotionStorage,
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

func cartCartTax(cart *Cart) int {
	total := 0
	for _, item := range cart.Items {
		total += item.TaxAmount * int(item.Quantity)
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
			s.handleCartChange(cart)
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
			s.handleCartChange(cart)
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
	s.handleCartChange(cart)
	err = s.saveCart(cart)
	if err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *DiskCartStorage) handleCartChange(cart *Cart) {

	if s.PromotionStorage != nil {
		available, err := s.PromotionStorage.GetPromotions()
		if err != nil {
			return
		}
		input := make([]*promotions.PromotionInput, 0)
		for _, cartItem := range cart.Items {
			input = append(input, cartItem.PromotionInput)
		}

		//all := append(input, item.PromotionInput)
		for _, promotion := range available {
			available := promotion.IsAvailable(input...)
			if available > 0 {
				outputs, err := promotion.Apply(input...)

				if err != nil {
					break
				}
				if hasPromotionApplied(cart.AppliedPromotions, promotion) >= available {
					continue
				}
				for _, output := range *outputs {

					for _, input := range input {
						if input.Sku == output.Sku {
							applyToItem(cart, output)

							break
						}
					}
				}
				if cart.AppliedPromotions == nil {
					cart.AppliedPromotions = make([]promotions.Promotion, 0)
				}
				cart.AppliedPromotions = append(cart.AppliedPromotions, promotion)
			}
		}
	}
	cart.TotalPrice = cartCartPrice(cart)
	cart.TaxAmount = cartCartTax(cart)
}

func applyToItem(cart *Cart, output promotions.PromotionOutput) {
	var found *CartItem = nil
	for _, item := range cart.Items {
		if item.Id == uint(output.Id) {
			found = &item
			break
		}
	}
	if found != nil {
		found.OriginalPrice = output.Price
		found.Price -= output.Discount
		found.TaxAmount = int(float64(output.Price) * 0.2)
	}
}

func hasPromotionApplied(applied []promotions.Promotion, output promotions.Promotion) int {
	ret := 0
	for _, p := range applied {
		if p.Id == output.Id {
			ret += 1
		}
	}
	return ret
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
