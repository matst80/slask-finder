package embeddings

import (
	"math"
	"sort"
	"strings"

	"tornberg.me/facet-search/pkg/types"
)

const (
	Price = iota
	Speed
	Portable
	New
	Used
	Fps
	Effective
	Features
	Accessory
	Popular
	Gaming
	Fun
	Eco
)

type MatchResult struct {
	Ids       types.ItemList
	SortIndex types.SortIndex
}

type Embeddings interface {
	AddDocument(doc EmbeddingsItem)
	RemoveDocument(id uint)
	FindMatches(embeddings []float64) MatchResult
}

type EmbeddingsItem struct {
	Embeddings []float64
	Id         uint
}

type EmbeddingsIndex struct {
	Documents []EmbeddingsItem
}

func NewEmbeddingsIndex() *EmbeddingsIndex {
	return &EmbeddingsIndex{
		Documents: make([]EmbeddingsItem, 0),
	}
}

func MakeDocument(item types.Item) EmbeddingsItem {
	return EmbeddingsItem{
		Embeddings: NormalizeEmbeddings(GetEmbeddingsForItem(item)),
		Id:         item.GetId(),
	}
}

func (i *EmbeddingsIndex) AddDocument(doc EmbeddingsItem) {
	i.Documents = append(i.Documents, doc)
}

func (i *EmbeddingsIndex) RemoveDocument(id uint) {
	for j, doc := range i.Documents {
		if doc.Id == id {
			i.Documents = append(i.Documents[:j], i.Documents[j+1:]...)
			return
		}
	}
}

func (i *EmbeddingsIndex) FindMatches(embeddings []float64) MatchResult {
	ret := make(types.ItemList, 0)
	sortMap := make(types.ByValue, 0)
	for _, doc := range i.Documents {
		similarity := CosineSimilarity(embeddings, doc.Embeddings)
		if similarity > 0.2 {
			ret.AddId(doc.Id)
			sortMap = append(sortMap, types.Lookup{Id: doc.Id, Value: similarity})
			//ret = append(ret, doc)
		}
	}
	sort.Sort(sort.Reverse(sortMap))
	sortIndex := make(types.SortIndex, len(sortMap))
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return MatchResult{
		Ids:       ret,
		SortIndex: sortIndex,
	}
}

const embeddingsLength = Eco + 1

func GetEmbedding(word string) []float64 {
	ret := make([]float64, embeddingsLength)
	words := strings.Split(strings.ToLower(word), " ")
	for _, w := range words {
		switch w {
		case "dyr":
			ret[Price] -= 0.8
		case "dyrt":
			ret[Price] -= 0.8
		case "bra":
			ret[Features] += 0.4
			ret[Price] += 0.2
		case "dålig":
			ret[Features] -= 0.4
			ret[Price] -= 0.2
		case "billig":
			ret[Price] += 0.8
		case "billigt":
			ret[Price] += 0.8
		case "prisvärd":
			ret[Price] += 0.8
			ret[Features] += 0.3
		case "prisvärt":
			ret[Price] += 0.8
			ret[Features] += 0.3
		case "snabb":
			ret[Speed] += 0.8
		case "portabel":
			ret[Portable] += 0.8
		case "ny":
			ret[New] += 0.8
		case "nytt":
			ret[New] += 0.8
		case "begagnad":
			ret[Used] += 0.8
		case "begagnat":
			ret[Used] += 0.8
		case "effektiv":
			ret[Effective] += 0.8
		case "funktioner":
			ret[Features] += 0.8
		case "miljövänlig":
			ret[Eco] += 0.8
		case "tillbehör":
			ret[Accessory] += 0.8
		case "populär":
			ret[Popular] += 0.8
		case "spel":
			ret[Gaming] += 0.8
		case "spela":
			ret[Gaming] += 0.8
		case "nvidia":
			ret[Gaming] += 0.4
		case "amd":
			ret[Gaming] += 0.2
		case "intel":
			ret[Gaming] += 0.2
		case "spelar":
			ret[Gaming] += 0.8
		case "rgb":
			ret[Gaming] += 0.4
		case "rolig":
			ret[Fun] += 0.8
		case "skal":
			ret[Accessory] += 0.4
		case "fodral":
			ret[Accessory] += 0.4
		case "väska":
			ret[Accessory] += 0.4
		case "till":
			ret[Accessory] += 0.4
		case "hz":
			ret[Fps] += 0.8
		case "fps":
			ret[Fps] += 0.8
		case "hertz":
			ret[Fps] += 0.8
		}
	}
	return ret
}

func GetEmbeddingsForItem(item types.Item) []float64 {
	ret := GetEmbedding(item.ToString())
	price := item.GetPrice()
	ret[Price] -= float64(price) / 10000000.0
	fields := item.GetFields()
	ret[Features] += (float64(len(fields)) / 50.0) - float64(0.5)
	if strings.Contains(item.GetTitle(), "MacBook") {
		ret[Price] += 0.8
		ret[Features] += 0.7
	}
	for id, value := range fields {
		if id == 10 {
			category := value.(string)
			if category == "Outlet" {
				ret[Price] += 0.1
				ret[Used] += 0.8
				ret[New] -= 0.8
			} else {
				ret[New] += 0.8
				ret[Used] -= 0.8
			}
			if category == "Gaming" {
				ret[Gaming] += 0.7
			}
		}
		if id == 8 {
			discount, ok := value.(int)
			if ok && discount > 0 {
				ret[Popular] += 0.1
				ret[Price] += float64(discount) / 100000.0
			}
		}
		if id == 6 {
			grade, ok := value.(int)
			if ok {
				ret[Popular] += (float64(grade) / 50) - 0.5
			}
		}
		if id == 20 {
			grade, ok := value.(string)
			if ok {
				if strings.Contains(grade, "skador") {
					ret[Used] += 0.8
					ret[New] -= 0.8
				}
			}
		}
		if id == 32240 {
			pcmark, ok := value.(int)
			if ok {
				ret[Speed] += float64(pcmark) / float64(price)
			}
		}
	}
	return ret
}

func NormalizeEmbeddings(embeddings []float64) []float64 {
	ret := make([]float64, embeddingsLength)
	max := 0.0
	for _, e := range embeddings {
		if e > max {
			max = e
		}
	}
	if max == 0 {
		return ret
	}
	for i, e := range embeddings {
		ret[i] = e / max
	}
	return ret
}

func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	dotProduct := 0.0
	magnitudeA := 0.0
	magnitudeB := 0.0
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		magnitudeA += a[i] * a[i]
		magnitudeB += b[i] * b[i]
	}
	magnitudeA = math.Sqrt(magnitudeA)
	magnitudeB = math.Sqrt(magnitudeB)
	if magnitudeA == 0 || magnitudeB == 0 {
		return 0
	}
	return dotProduct / (magnitudeA * magnitudeB)
}
