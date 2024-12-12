package index

import (
	"encoding/json"
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
	return i.Name + " " + i.Description
}

type StoreContentItem struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Lat         string `json:"lat"`
	Lng         string `json:"lng"`
}

func (i StoreContentItem) GetId() uint {
	return i.Id
}

func (i StoreContentItem) IndexData() string {
	return i.Name + " " + i.Description
}

func ContentItemFromLine(record []string) (ContentItem, error) {
	if record[StoreID] == "" {
		idString := record[Id]
		if strings.HasPrefix(idString, "SELLER:") {
			cleanId := strings.Replace(idString, "SELLER:", "", -1)
			id, err := strconv.Atoi(cleanId)
			if err != nil {
				return nil, err
			}
			var picture interface{}
			if record[PagePictureUrl] != "" {
				json.Unmarshal([]byte(record[PagePictureUrl]), &picture)
			}
			return SellerContentItem{
				Id:          uint(id),
				Name:        record[PageTitle],
				Description: record[PageDetailText],
				Url:         record[PageUrl],
				Picture:     picture,
				//Component:   component,
			}, nil
		}
		cleanId := strings.Replace(idString, "contentbean:", "", -1)
		id, err := strconv.Atoi(cleanId)
		if err != nil {
			return nil, err
		}
		var picture interface{}
		if record[PagePictureUrl] != "" {
			json.Unmarshal([]byte(record[PagePictureUrl]), &picture)
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
			Url:         record[PageUrl],
			Picture:     picture,
			//Component:   component,
		}, nil
	} else {
		id, err := strconv.Atoi(record[StoreID])
		if err != nil {
			return nil, err
		}
		return StoreContentItem{
			Id:          uint(id),
			Name:        record[PageTitle],
			Description: record[StoreAddress],
			Image:       record[ComponentsPictures],
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

func (i *ContentIndex) MatchQuery(query string) []ContentItem {
	result := i.Search.Search(query)
	//sortResult := make(chan *types.SortIndex)
	//result.GetSorting(sortResult)
	//defer close(sortResult)
	//s := <-sortResult
	itemIds := *result.ToResult()
	j := min(30, len(itemIds))
	resultItems := make([]ContentItem, 0, j)

	for id := range *result.ToResult() {
		item, ok := i.Items[id]
		if ok {
			resultItems = append(resultItems, item)
			j--
		}
		if j == 0 {
			break
		}
	}
	return resultItems
}
