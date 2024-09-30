package promotions

import (
	"encoding/json"
	"os"
)

type PromotionStorage interface {
	GetPromotions() ([]Promotion, error)
	AddPromotion(promotion Promotion) error
	RemovePromotion(id int) error
}

type DiskPromotionStorage struct {
	Path string
}

type PromotionFile struct {
	Promotions []Promotion `json:"promotions"`
}

func (s *DiskPromotionStorage) readFile(path string, dest any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(dest)
	return err
}

func (s *DiskPromotionStorage) writeFile(path string, src any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	err = json.NewEncoder(file).Encode(src)
	return err
}

func (s *DiskPromotionStorage) GetPromotions() ([]Promotion, error) {
	file := PromotionFile{}
	err := s.readFile(s.Path, &file)
	if err != nil {
		return nil, err
	}
	return file.Promotions, nil
}

func (s *DiskPromotionStorage) AddPromotion(promotion Promotion) error {
	existing, err := s.GetPromotions()
	if err != nil {
		return err
	}
	for _, p := range existing {
		if p.Id == promotion.Id {
			return nil
		}
	}
	file := PromotionFile{
		Promotions: append(existing, promotion),
	}
	return s.writeFile(s.Path, file)
}

func (s *DiskPromotionStorage) RemovePromotion(id int) error {
	existing, err := s.GetPromotions()
	if err != nil {
		return err
	}
	newPromotions := make([]Promotion, 0)
	for _, p := range existing {
		if p.Id != id {
			newPromotions = append(newPromotions, p)
		}
	}
	file := PromotionFile{
		Promotions: newPromotions,
	}
	return s.writeFile(s.Path, file)
}
