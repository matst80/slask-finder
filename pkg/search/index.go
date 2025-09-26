package search

import (
	"log"
	"maps"
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
		var prev Token
		var hasPrev bool
		i.tokenizer.Tokenize(property, func(token Token, original string, _ int, last bool) bool {
			if j == 0 {
				i.Trie.Insert(token, original, id)
				// Record bigram transitions from the same field that feeds the Trie
				if hasPrev {
					i.Trie.AddTransition(prev, token)
				}
				prev = token
				hasPrev = true
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

// FindTrieMatchesForContext finds matches for the last word and ranks them
// using the previous token (if provided) via the Trie's Markov chain.
func (a *FreeTextIndex) FindTrieMatchesForContext(prevWord string, word string, resultChan chan<- []Match) {
	prefix := NormalizeWord(word)
	if len(prefix) == 0 {
		resultChan <- []Match{}
		return
	}
	prev := NormalizeWord(prevWord)
	if len(prev) == 0 {
		resultChan <- a.Trie.FindMatches(prefix)
		return
	}
	resultChan <- a.Trie.FindMatchesWithPrev(prefix, prev)
}

func (i *FreeTextIndex) Lock() {
	i.mu.Lock()
}

func (i *FreeTextIndex) Unlock() {
	i.mu.Unlock()
}

func (i *FreeTextIndex) RemoveDocument(id uint) {
	for token := range i.TokenMap {
		if ids, ok := i.TokenMap[token]; ok {
			delete(*ids, id)
		}
	}
	maps.DeleteFunc(i.TokenMap, func(_ Token, ids *types.ItemList) bool {
		return len(*ids) == 0
	})
	// i.Trie.RemoveDocument(id)
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
	for i := range i.TokenMap {
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

// TODO maybe two itemlists, one for exact and one for fuzzy

func (i *FreeTextIndex) Filter(query string, res *types.ItemList) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	mappings := types.CurrentSettings.WordMappings

	i.tokenizer.Tokenize(query, func(token Token, original string, count int, last bool) bool {
		log.Printf("filter on token %s", token)
		ids, found := i.TokenMap[token]
		if found && ids != nil {
			if res.HasIntersection(ids) {
				res.Intersect(*ids)
			} else {
				found = false
			}
		}
		if word, ok := mappings[string(token)]; ok {
			ids, found = i.TokenMap[Token(word)]
			if res.HasIntersection(ids) {
				res.Intersect(*ids)
				found = true
			}
		}

		if !found {
			tries := types.ItemList{}
			for _, match := range i.Trie.FindMatches(token) {
				if match.Items != nil {
					if res.HasIntersection(match.Items) {
						tries.Merge(match.Items)
					}
					found = true
				}
			}

			// fuzzy
			fuzzyMatches := i.getBestFuzzyMatch(token, 3)
			for _, match := range fuzzyMatches {
				if fuzzyIds, ok := i.TokenMap[match]; ok {
					if fuzzyIds != nil {
						if res.HasIntersection(fuzzyIds) {
							tries.Merge(fuzzyIds)
						}
					}
				}
			}
			res.Intersect(tries)
		}

		return len(*res) > 0
	})
}

func (i *FreeTextIndex) Search(query string) *types.ItemList {
	res := &types.ItemList{}
	//mergeLimit := types.CurrentSettings.SearchMergeLimit
	i.mu.RLock()
	defer i.mu.RUnlock()

	mappings := types.CurrentSettings.WordMappings

	i.tokenizer.Tokenize(query, func(token Token, original string, count int, last bool) bool {
		ids, found := i.TokenMap[token]
		if found {
			if count == 0 {
				res.Merge(ids)
			} else if res.HasIntersection(ids) {
				res.Intersect(*ids)
			} else {
				found = false
			}
		}
		if word, ok := mappings[string(token)]; ok {
			ids, found = i.TokenMap[Token(word)]
			if !found || count == 0 {
				res.Merge(ids)
			} else if res.HasIntersection(ids) {
				res.Intersect(*ids)
			} else {
				found = false
			}
		}
		foundTrie := found
		//log.Printf("word: %s, found: %v, last: %v", token, found, last)
		if !found {
			for _, match := range i.Trie.FindMatches(token) {
				foundTrie = true
				res.Merge(match.Items)
			}
		}
		if !foundTrie {
			// fuzzy
			fuzzyMatches := i.getBestFuzzyMatch(token, 3)
			for _, match := range fuzzyMatches {
				if fuzzyIds, ok := i.TokenMap[match]; ok {
					res.Merge(fuzzyIds)
				}
			}
		}

		return len(*res) > 0
	})

	return res

}
