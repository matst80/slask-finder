package search

import (
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

type FreeTextIndex struct {
	mu        sync.RWMutex
	tokenizer *Tokenizer
	Trie      *Trie
	//Documents map[uint]*Document
	TokenMap map[Token]*types.ItemList
	//BaseSortMap map[uint]float64
	WordMappings map[Token]Token
	//Tokens []Token
}

//type DocumentResult map[uint]float64

// func (i *FreeTextIndex) AddDocument(doc *Document) {
// 	i.mu.Lock()
// 	defer i.mu.Unlock()
// 	//i.Documents[doc.Id] = doc
// 	for _, token := range doc.Tokens {
// 		if _, ok := i.TokenMap[token]; !ok {
// 			i.TokenMap[token] = make([]*Document, 0)
// 			i.Tokens = append(i.Tokens, token)
// 		}
// 		i.TokenMap[token] = append(i.TokenMap[token], doc)
// 	}
// }

func (i *FreeTextIndex) CreateDocument(id uint, text ...string) {

	i.mu.Lock()
	defer i.mu.Unlock()
	i.CreateDocumentUnsafe(id, text...)
}

func (i *FreeTextIndex) CreateDocumentUnsafe(id uint, text ...string) {
	//i.tokenizer.MakeDocument(id, text...)

	for j, property := range text {
		i.tokenizer.Tokenize(property, func(token Token, original string) bool {
			if j == 0 {
				i.Trie.Insert(token, original, id)
			}
			if l, ok := i.TokenMap[token]; !ok {
				i.TokenMap[token] = &types.ItemList{id: struct{}{}}
			} else {
				l.AddId(id)
			}
			return true
		})
	}
}

func (a *FreeTextIndex) FindTrieMatchesForWord(word string, resultChan chan<- []Match) {
	token := NormalizeWord(word)
	if len(token) == 0 {
		resultChan <- []Match{}
		return
	}
	resultChan <- a.Trie.FindMatches(token)
}

func (i *FreeTextIndex) Lock() {
	i.mu.Lock()
}

func (i *FreeTextIndex) Unlock() {
	i.mu.Unlock()
}

func (i *FreeTextIndex) RemoveDocument(id uint, text ...string) {
	//delete(i.Documents, id)
}

func NewFreeTextIndex(tokenizer *Tokenizer) *FreeTextIndex {
	return &FreeTextIndex{
		tokenizer:    tokenizer,
		TokenMap:     map[Token]*types.ItemList{},
		WordMappings: make(map[Token]Token),
		Trie:         NewTrie(),
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

// func absMin(x, y int) int {
// 	if x < y {
// 		return x
// 	}
// 	return y
// }

func (i *FreeTextIndex) getBestFuzzyMatch(token Token, max int) []Token {
	matching := make([]tokenScore, max)
	for j := 0; j < max; j++ {
		matching[j] = tokenScore{score: -99999999.0, token: token}
	}
	tl := len(token)

	score := 0.0
	found := false
	for i, _ := range i.TokenMap {
		il := len(i)
		if il < tl {
			continue
		}
		score = 0.0
		found = false
		for _, chr := range token {
			found = false
			for _, jchr := range i {
				if chr == jchr {
					score += 4.0
					found = true
					break
				}
			}
			if !found {
				score -= float64(tl)
			}
		}
		score -= float64(absDiffInt(il, tl))
		for j := 0; j < max; j++ {
			if matching[j].score < score {
				matching[j].score = score
				matching[j].token = i
				break
			}
		}
	}
	ret := make([]Token, 0, max)
	for j := 0; j < max; j++ {
		if matching[j].score < 0 {
			break
		}
		ret = append(ret, matching[j].token)
	}
	return ret
	// slices.SortFunc(matching, func(i, j tokenScore) int {
	// 	return cmp.Compare(j.score, i.score)
	// })
	// max := absMin(len(matching), 5)
	// res := make([]Token, max)
	// for idx, match := range matching {
	// 	if idx >= max {
	// 		break
	// 	}
	// 	res[idx] = match.token
	// }
	// return res
}

// func (i *FreeTextIndex) getMatchDocs(tokens []Token) *types.ItemList {

// 	//res := make(map[uint]*Document)
// 	missingStrings := make([]Token, 0)
// 	res := &types.ItemList{}
// 	for j, token := range tokens {
// 		docs, ok := i.TokenMap[token]
// 		if ok {
// 			if j == 0 {
// 				res.Merge(docs)
// 			} else {
// 				res.Intersect(*docs)
// 			}
// 		} else {
// 			missingStrings = append(missingStrings, i.getBestFuzzyMatch(token, 3)...)
// 		}
// 	}
// 	for _, token := range missingStrings {
// 		docs, ok := i.TokenMap[token]
// 		if ok {
// 			copy := &types.ItemList{}
// 			res.Merge(docs)
// 			if len(*copy) == 0 {
// 				copy.Merge(docs)
// 			} else {
// 				copy.Intersect(*docs)
// 			}
// 			if len(*copy) > 0 {
// 				return copy
// 			}
// 		}
// 	}

// 	return res
// }

func (i *FreeTextIndex) Search(query string) *types.ItemList {
	res := &types.ItemList{}
	first := true
	i.mu.RLock()
	defer i.mu.RUnlock()
	i.tokenizer.Tokenize(query, func(token Token, original string) bool {
		ids, found := i.TokenMap[token]
		if found {
			if first {
				res.Merge(ids)
				first = false
			} else if res.HasIntersection(ids) {
				res.Intersect(*ids)
			} else {
				found = false
			}
		}

		if !found {
			// fuzzy or trie
			for j, match := range i.Trie.FindMatches(token) {
				if len(*match.Items) > 0 {
					if first {
						res.Merge(match.Items)
						first = false
						found = true
					} else if res.HasIntersection(match.Items) {
						res.Intersect(*match.Items)
						found = true
						break
					}
				}
				if j > 50 {
					break
				}
			}
		}
		if !found {
			// fuzzy
			fuzzyMatches := i.getBestFuzzyMatch(token, 3)
			for _, match := range fuzzyMatches {
				if _, ok := i.TokenMap[match]; ok {
					if first {
						res.Merge(i.TokenMap[match])
						first = false
					} else if res.HasIntersection(i.TokenMap[match]) {
						res.Intersect(*i.TokenMap[match])
						break
					}
				}
			}
		}

		return len(*res) > 0
	})

	return res
	// score := 0.0
	// wordIdx := 0
	// lastIndex := 0
	// var missing int
	// var word Token
	// for _, doc := range result {
	// 	score = 50.0
	// 	wordIdx = 0
	// 	lastIndex = 0
	// 	missing = len(tokens)
	// 	word = tokens[wordIdx]
	// 	for i, t := range doc.Tokens {
	// 		if word == t {
	// 			score += max(1, 20.0-(2.0*float64(absDiffInt(i, lastIndex))))
	// 			lastIndex = i
	// 			wordIdx++
	// 			missing--
	// 			if wordIdx >= len(tokens) {
	// 				break
	// 			}
	// 			word = tokens[wordIdx]
	// 		}
	// 	}
	// 	res[doc.Id] = score - float64(missing)*0.2
	// 	// if res[doc.Id] > 0 {
	// 	// 	//l := float64(len(tokens))
	// 	// 	//dl := float64(len(doc.Tokens))
	// 	// 	// base := 0.0
	// 	// 	// if i.BaseSortMap != nil {
	// 	// 	// 	if v, ok := i.BaseSortMap[doc.Id]; ok {
	// 	// 	// 		base = v
	// 	// 	// 	}
	// 	// 	// }
	// 	// 	hits := res[doc.Id]
	// 	// 	res[doc.Id] = (hits * 1000.0)
	// 	// }
	// }

	// return &res
}

// func (d *DocumentResult) ToSortIndex() []types.Lookup {
// 	// l := len(*d)

// 	// sortMap := make(types.ByValue, l)
// 	// idx := 0
// 	// for id, score := range *d {
// 	// 	sortMap[idx] = types.Lookup{Id: id, Value: score}
// 	// 	idx++
// 	// }
// 	return slices.SortedFunc(func(yield func(types.Lookup) bool) {
// 		for id, score := range *d {
// 			if !yield(types.Lookup{Id: id, Value: score}) {
// 				break
// 			}
// 		}
// 	}, types.LookUpReversed)

// }

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

// type ResultWithSort struct {
// 	*types.IdList
// 	SortIndex types.SortIndex
// }

// func (d *DocumentResult) ToResult() *types.ItemList {
// 	res := types.ItemList{}

// 	for id := range *d {
// 		res[id] = struct{}{}
// 	}
// 	return &res
// }

// func (d *DocumentResult) IntersectTo(items *types.ItemList) {
// 	for id := range *items {
// 		if _, ok := (*d)[id]; !ok {
// 			delete(*items, id)
// 		}
// 	}
// }

// func (d *DocumentResult) GetSorting(sortChan chan<- *types.ByValue) {
// 	v := types.ByValue(d.ToSortIndex())
// 	sortChan <- &v
// }

// func (d *DocumentResult) GetSortingWithAdditionalItems(idList *types.ItemList, sortMap map[uint]float64, sortChan chan<- *types.ByValue) {
// 	sortChan <- d.ToSortIndexWithAdditionalItems(idList, sortMap)
// }
