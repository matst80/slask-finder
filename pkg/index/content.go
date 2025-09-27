package index

import (
	"encoding/json"
	"errors"
	"iter"
	"log"
	"strconv"
	"strings"

	"github.com/matst80/slask-finder/pkg/search"
)

type ContentItem interface {
	GetId() uint
	IndexData() string
}

const (
	Rank = iota
	Score
	DQAttributes
	FFCheckoutCount
	StoreAddress
	StoreCapabilities
	StoreCity
	StoreFeatures
	StoreID
	StoreOpeningHours
	StorePhonenumber
	StoreShortName
	StoreEMailAddress
	StoreLatitude
	StoreLongitude
	ComponentDetailText
	ComponentKeywordTags
	ComponentSeoDescriptions
	ComponentSeoKeywords
	ComponentSubjectTags
	ComponentTeaserTexts
	ComponentTeaserTitles
	ComponentTitles
	ComponentsPictures
	CreationDate
	FeederState
	FeederTime
	Id
	IsDeleted
	KeywordTags
	ModificationDate
	PageDetailText
	PageDisplayDate
	PageLocale
	PagePictureUrl
	PageTargetGroup
	PageTeaserTitle
	PageTitle
	PageType
	PageUrl
	SeoDescription
	SeoKeywords
	SeoTitle
	SubjectTags
	ValidFrom
	ValidTo
)

type CmsComponent struct {
	DetailTest string      `json:"detailText"`
	TeaserText string      `json:"teaserText"`
	Tiles      interface{} `json:"tiles"`
	Pictures   interface{} `json:"pictures"`
}

type CmsContentItem struct {
	Id          uint        `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Picture     interface{} `json:"picture"`
	Features    string      `json:"features"`
	PhoneNumber string      `json:"phoneNumber"`
	//Component   *CmsComponent `json:"component,omitempty"`
	Url string `json:"url"`
}

type SellerContentItem struct {
	Id          uint        `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Image       string      `json:"image"`
	Picture     interface{} `json:"picture"`
	Url         string      `json:"url"`
}

func (i SellerContentItem) GetId() uint {
	return i.Id
}

func (i SellerContentItem) IndexData() string {
	return i.Name + " " + i.Description
}

func (i CmsContentItem) GetId() uint {
	return i.Id
}

func (i CmsContentItem) IndexData() string {
	return i.Name
}

type StoreContentItem struct {
	Id          uint        `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Image       string      `json:"image"`
	OpenHours   interface{} `json:"openHours"`
	Url         string      `json:"url"`
	Lat         string      `json:"lat"`
	Lng         string      `json:"lng"`
}

func (i StoreContentItem) GetId() uint {
	return i.Id
}

func (i StoreContentItem) IndexData() string {
	return i.Name + " " + i.Description
}

func fixUrl(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return strings.Replace(url, "/elgiganten-se-sv", "https://www.elgiganten.se", 1)
}

func ContentItemFromLine(record []string) (ContentItem, error) {
	// log.Printf("Record deleted: %v", record[IsDeleted])
	if record[IsDeleted] == "true" {
		return nil, errors.New("item is deleted")
	}
	if record[StoreID] == "" {
		idString := record[Id]
		if strings.HasPrefix(idString, "SELLER:") {
			cleanId := strings.Replace(idString, "SELLER:", "", -1)
			id, err := strconv.ParseUint(cleanId, 10, 64)
			if err != nil {
				return nil, err
			}
			var picture interface{}
			if record[PagePictureUrl] != "" {
				if err := json.Unmarshal([]byte(record[PagePictureUrl]), &picture); err != nil {
					log.Printf("Could not unmarshal PagePictureUrl: %v", err)
				}
			}
			return SellerContentItem{
				Id:          uint(id),
				Name:        record[PageTitle],
				Description: record[PageDetailText],
				Url:         fixUrl(record[PageUrl]),
				Picture:     picture,
				//Component:   component,
			}, nil
		}
		cleanId := strings.Replace(idString, "contentbean:", "", -1)
		id, err := strconv.ParseUint(cleanId, 10, 64)
		if err != nil {
			return nil, err
		}
		var picture interface{}
		if record[PagePictureUrl] != "" {
			if err := json.Unmarshal([]byte(record[PagePictureUrl]), &picture); err != nil {
				log.Printf("Could not unmarshal PagePictureUrl: %v", err)
			}
		}
		//var component *CmsComponent
		// if record[ComponentsPictures] != "" {
		// 	var tiles interface{}
		// 	var pictures interface{}
		// 	json.Unmarshal([]byte(record[ComponentTitles]), &tiles)
		// 	json.Unmarshal([]byte(record[ComponentsPictures]), &pictures)
		// 	component = &CmsComponent{
		// 		DetailTest: record[ComponentDetailText],
		// 		TeaserText: record[ComponentTeaserTexts],
		// 		Tiles:      tiles,
		// 		Pictures:   tiles,
		// 	}
		// }
		return CmsContentItem{
			Id:          uint(id),
			Name:        record[PageTitle],
			Description: record[PageDetailText],
			Url:         fixUrl(record[PageUrl]),
			Picture:     picture,
			//Component:   component,
		}, nil
	} else {
		id, err := strconv.ParseUint(record[StoreID], 10, 64)
		if err != nil {
			return nil, err
		}
		var openHours interface{}
		if record[StoreOpeningHours] != "" {
			if err := json.Unmarshal([]byte(record[StoreOpeningHours]), &openHours); err != nil {
				log.Printf("Could not unmarshal StoreOpeningHours: %v", err)
			}
		}
		return StoreContentItem{
			Id:          uint(id),
			Name:        record[PageTitle],
			Description: record[StoreAddress],
			Image:       record[ComponentsPictures],
			OpenHours:   openHours,
			Url:         fixUrl(record[PageUrl]),
			Lat:         record[StoreLatitude],
			Lng:         record[StoreLongitude],
		}, nil
	}
	//return nil, errors.New("Unknown content type")
}

type ContentIndex struct {
	Items  map[uint]ContentItem
	Search *search.FreeTextIndex
}

func NewContentIndex() *ContentIndex {
	return &ContentIndex{
		Items:  make(map[uint]ContentItem, 0),
		Search: search.NewFreeTextIndex(&search.Tokenizer{MaxTokens: 128}),
	}
}

func (i *ContentIndex) AddItem(item ContentItem) {
	i.Items[item.GetId()] = item
	i.Search.CreateDocument(item.GetId(), item.IndexData())
}

func (i *ContentIndex) MatchQuery(query string) iter.Seq[ContentItem] {
	result := i.Search.Search(query)

	return func(yield func(ContentItem) bool) {
		for id := range *result {
			item, ok := i.Items[id]
			if ok {
				if !yield(item) {
					break
				}
			}
		}
	}

}
