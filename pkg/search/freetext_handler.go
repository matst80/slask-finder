package search

import (
	"log"
	"maps"
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

type FreeTextItemHandler struct {
	mu           sync.RWMutex
	tokenizer    *Tokenizer
	Trie         *Trie
	TokenMap     map[Token]*types.ItemList
	WordMappings map[Token]Token
	All          types.ItemList
}

type FreeTextItemHandlerOptions struct {
	Tokenizer *Tokenizer
}

func DefaultFreeTextHandlerOptions() FreeTextItemHandlerOptions {
	return FreeTextItemHandlerOptions{
		Tokenizer: &Tokenizer{MaxTokens: 128},
	}
}

func NewFreeTextItemHandler(opts FreeTextItemHandlerOptions) *FreeTextItemHandler {
	handler := &FreeTextItemHandler{
		mu:           sync.RWMutex{},
		tokenizer:    opts.Tokenizer,
		Trie:         NewTrie(),
		TokenMap:     make(map[Token]*types.ItemList),
		WordMappings: make(map[Token]Token),
		All:          make(types.ItemList),
	}

	return handler
}

func (h *FreeTextItemHandler) HandleItem(item types.Item, wg *sync.WaitGroup) {

	wg.Go(func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		id := item.GetId()
		if item.IsDeleted() {
			h.RemoveDocument(id)
			delete(h.All, id)
			// h.Trie.RemoveDocument(id)
		} else {
			if _, exists := h.All[id]; !exists {
				h.All[id] = struct{}{}
				h.CreateDocumentUnsafe(id, item.ToStringList()...)
			}
		}
	})
}

func (i *FreeTextItemHandler) RemoveDocument(id uint) {
	for token := range i.TokenMap {
		if ids, ok := i.TokenMap[token]; ok {
			delete(*ids, id)
		}
	}
	maps.DeleteFunc(i.TokenMap, func(_ Token, ids *types.ItemList) bool {
		return len(*ids) == 0
	})
}

func (i *FreeTextItemHandler) CreateDocumentUnsafe(id uint, text ...string) {

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

func (h *FreeTextItemHandler) MatchQuery(query string, qm *types.QueryMerger) {
	if query == "" {
		return
	}
	if query == "*" {
		qm.Add(func() *types.ItemList {
			clone := maps.Clone(h.All)
			return &clone
		})
	} else {
		qm.Add(func() *types.ItemList {
			return h.Search(query)
		})
	}
}

func (i *FreeTextItemHandler) getBestFuzzyMatch(token Token, max int) []Token {
	matching := make([]tokenScore, max)
	for j := range max {
		matching[j] = tokenScore{score: -99999999.0, token: token}
	}
	tl := len(token)

	score := 0.0
	for i := range i.TokenMap {
		il := len(i)
		if il < tl {
			continue
		}
		score = 0.0
		found := false
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
		for j := range max {
			if matching[j].score < score {
				matching[j].score = score
				matching[j].token = i
				break
			}
		}
	}
	ret := make([]Token, 0, max)
	for j := range max {
		if matching[j].score < 0 {
			break
		}
		ret = append(ret, matching[j].token)
	}
	return ret
}

// TODO maybe two itemlists, one for exact and one for fuzzy

func (i *FreeTextItemHandler) Filter(query string, res *types.ItemList) {
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
				if match.Items != nil && res.HasIntersection(match.Items) {
					tries.Merge(match.Items)
				}
			}

			// fuzzy
			fuzzyMatches := i.getBestFuzzyMatch(token, 3)
			for _, match := range fuzzyMatches {
				if fuzzyIds, ok := i.TokenMap[match]; ok && fuzzyIds != nil && res.HasIntersection(fuzzyIds) {
					tries.Merge(fuzzyIds)
				}
			}
			res.Intersect(tries)
		}

		return len(*res) > 0
	})
}

func (i *FreeTextItemHandler) Search(query string) *types.ItemList {
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

func (a *FreeTextItemHandler) FindTrieMatchesForWord(word string, resultChan chan<- []Match) {
	token := NormalizeWord(word)
	if len(token) == 0 {
		resultChan <- []Match{}
		return
	}
	resultChan <- a.Trie.FindMatches(token)
}

func (a *FreeTextItemHandler) FindTrieMatchesForContext(prevWord string, word string, resultChan chan<- []Match) {
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
