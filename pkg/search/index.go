package search

import (
	"cmp"
	"slices"
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

type FreeTextIndex struct {
	mu        sync.RWMutex
	Tokenizer *Tokenizer
	Documents map[uint]*Document
	TokenMap  map[Token][]*Document
	//BaseSortMap map[uint]float64
	Tokens []string
}

type DocumentResult map[uint]float64

func (i *FreeTextIndex) AddDocument(doc *Document) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Documents[doc.Id] = doc
	for _, token := range doc.Tokens {
		if _, ok := i.TokenMap[token]; !ok {
			i.TokenMap[token] = make([]*Document, 0)
			i.Tokens = append(i.Tokens, string(token))
		}
		i.TokenMap[token] = append(i.TokenMap[token], doc)
	}
}

func (i *FreeTextIndex) CreateDocument(id uint, text ...string) {
	i.AddDocument(i.Tokenizer.MakeDocument(id, text...))
}

func (i *FreeTextIndex) RemoveDocument(id uint) {
	delete(i.Documents, id)
}

func NewFreeTextIndex(tokenizer *Tokenizer) *FreeTextIndex {
	return &FreeTextIndex{
		Tokenizer: tokenizer,
		Documents: make(map[uint]*Document),
		TokenMap:  map[Token][]*Document{},
		Tokens:    make([]string, 0),
	}
}

type tokenScore struct {
	score float64
	token Token
}

func absDiffInt(x, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}

func absMin(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (i *FreeTextIndex) getRankedFuzzyMatch(token string) []Token {
	matching := make([]tokenScore, 0)
	tl := len(token)
	score := 0.0
	found := false
	for _, i := range i.Tokens {
		il := len(i)
		if il < tl {
			continue
		}
		score = 0.0
		found = false
		for idx, chr := range token {
			found = false
			for jdx, jchr := range i {
				if chr == jchr {
					score += float64(tl-absDiffInt(idx, jdx)) * 4.0
					found = true
					break
				}
			}
			if !found {
				score -= float64(tl)
			}
		}
		score -= float64(absDiffInt(il, tl) * 2)
		if score > float64(tl)*2.0 {
			matching = append(matching, tokenScore{score: score, token: Token(i)})
		}
	}
	slices.SortFunc(matching, func(i, j tokenScore) int {
		return cmp.Compare(j.score, i.score)
	})
	max := absMin(len(matching), 5)
	res := make([]Token, max)
	for idx, match := range matching {
		if idx >= max {
			break
		}
		res[idx] = match.token
	}
	return res
}

func (i *FreeTextIndex) getMatchDocs(tokens []Token) map[uint]*Document {

	res := make(map[uint]*Document)
	missingStrings := make([]string, 0)
	for _, token := range tokens {
		docs, ok := i.TokenMap[token]
		if ok {
			for _, doc := range docs {
				res[doc.Id] = doc
			}
		} else {
			missingStrings = append(missingStrings, string(token))
		}

	}

	for _, token := range missingStrings {
		matches := i.getRankedFuzzyMatch(token)
		for _, match := range matches {
			if docs, ok := i.TokenMap[match]; ok {

				for _, doc := range docs {
					res[doc.Id] = doc
				}

			}
		}
	}

	return res
}

func (i *FreeTextIndex) Search(query string) *DocumentResult {

	tokens := i.Tokenizer.Tokenize(query)
	res := make(DocumentResult)
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := i.getMatchDocs(tokens)
	score := 0.0
	wordIdx := 0
	lastIndex := 0
	var missing int
	var word Token
	for _, doc := range result {
		score = 50.0
		wordIdx = 0
		lastIndex = 0
		missing = len(tokens)
		word = tokens[wordIdx]
		for i, t := range doc.Tokens {
			if word == t {
				score += max(1, 100.0-(10.0*float64(absDiffInt(i, lastIndex))))
				lastIndex = i
				wordIdx++
				missing--
				if wordIdx >= len(tokens) {
					break
				}
				word = tokens[wordIdx]
			}
		}
		res[doc.Id] = score - float64(missing*30)
		// if res[doc.Id] > 0 {
		// 	//l := float64(len(tokens))
		// 	//dl := float64(len(doc.Tokens))
		// 	// base := 0.0
		// 	// if i.BaseSortMap != nil {
		// 	// 	if v, ok := i.BaseSortMap[doc.Id]; ok {
		// 	// 		base = v
		// 	// 	}
		// 	// }
		// 	hits := res[doc.Id]
		// 	res[doc.Id] = (hits * 1000.0)
		// }
	}

	return &res
}

func (d *DocumentResult) ToSortIndex() []types.Lookup {
	// l := len(*d)

	// sortMap := make(types.ByValue, l)
	// idx := 0
	// for id, score := range *d {
	// 	sortMap[idx] = types.Lookup{Id: id, Value: score}
	// 	idx++
	// }
	return slices.SortedFunc(func(yield func(types.Lookup) bool) {
		for id, score := range *d {
			if !yield(types.Lookup{Id: id, Value: score}) {
				break
			}
		}
	}, types.LookUpReversed)

}

// func (d *DocumentResult) ToSortIndexWithAdditionalItems(additionalIds *types.ItemList, baseMap map[uint]float64) *types.ByValue {

// 	l := len(*d)

// 	if (*additionalIds) != nil {
// 		l += len(*additionalIds)
// 	}

// 	sortMap := make(types.ByValue, l)
// 	idx := 0
// 	for id, score := range *d {
// 		sortMap[idx] = types.Lookup{Id: id, Value: score}
// 		idx++
// 	}
// 	if (*additionalIds) != nil {
// 		for id := range *additionalIds {
// 			if _, ok := (*d)[id]; !ok {
// 				sortMap[idx] = types.Lookup{Id: id, Value: baseMap[id]}
// 				idx++
// 			}

// 		}
// 	}
// 	sort.Sort(sort.Reverse(sortMap[:idx]))

// 	return &sortMap
// }

// func (d *DocumentResult) ToSortIndexAll(inputMap map[uint]float64) *types.SortIndex {
// 	l := len(inputMap)

// 	sortMap := make(types.ByValue, l)
// 	copy(sortMap, types.ByValue{})
// 	idx := 0
// 	for id, score := range *d {
// 		sortMap[idx] = types.Lookup{Id: id, Value: score + inputMap[id]}
// 		idx++
// 	}
// 	sort.Sort(sort.Reverse(sortMap))
// 	sortIndex := make(types.SortIndex, l)
// 	for idx, item := range sortMap {
// 		sortIndex[idx] = item.Id
// 	}
// 	return &sortIndex
// }

type ResultWithSort struct {
	*types.IdList
	SortIndex types.SortIndex
}

func (d *DocumentResult) ToResult() *types.ItemList {
	res := types.ItemList{}

	for id := range *d {
		res[id] = struct{}{}
	}
	return &res
}

func (d *DocumentResult) IntersectTo(items *types.ItemList) {
	for id := range *items {
		if _, ok := (*d)[id]; !ok {
			delete(*items, id)
		}
	}
}

// func (d *DocumentResult) GetSorting(sortChan chan<- *types.ByValue) {
// 	v := types.ByValue(d.ToSortIndex())
// 	sortChan <- &v
// }

// func (d *DocumentResult) GetSortingWithAdditionalItems(idList *types.ItemList, sortMap map[uint]float64, sortChan chan<- *types.ByValue) {
// 	sortChan <- d.ToSortIndexWithAdditionalItems(idList, sortMap)
// }
