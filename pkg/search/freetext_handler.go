package search

import (
	"log"
	"sync"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalItems = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "slaskfinder_items_total",
		Help: "The total number of items in index",
	})
)

type FreeTextItemHandler struct {
	mu           sync.RWMutex
	tokenizer    *Tokenizer
	Trie         *Trie
	TokenMap     map[Token]*roaring.Bitmap
	WordMappings map[Token]Token
	All          *types.ItemList
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
		TokenMap:     make(map[Token]*roaring.Bitmap),
		WordMappings: make(map[Token]Token),
		All:          types.NewItemList(),
	}

	return handler
}

func (h *FreeTextItemHandler) HandleItem(item types.Item, wg *sync.WaitGroup) {

	wg.Go(func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		itemId := item.GetId()
		id := uint32(itemId)
		exists := h.All.Contains(id)
		if item.IsDeleted() {
			if exists {
				h.RemoveDocument(itemId)
			}
			h.All.RemoveId(id)

			// h.Trie.RemoveDocument(id)
		} else {

			if !exists {
				h.All.AddId(id)

				h.CreateDocumentUnsafe(itemId, item.ToStringList()...)
			}
		}

	})
}

func (i *FreeTextItemHandler) RemoveDocument(itemId types.ItemId) {
	id := uint32(itemId)
	tokensToDelete := make([]Token, 0)
	for token := range i.TokenMap {
		if ids, ok := i.TokenMap[token]; ok {
			if ids.Contains(id) {
				ids.Remove(id)
				if ids.IsEmpty() {
					tokensToDelete = append(tokensToDelete, token)
				}
			}
			//delete(*ids, id)
		}
	}
	if len(tokensToDelete) > 0 {
		for _, token := range tokensToDelete {
			delete(i.TokenMap, token)
		}
	}
}

func (i *FreeTextItemHandler) CreateDocumentUnsafe(id types.ItemId, text ...string) {

	for j, property := range text {
		var prev Token
		var hasPrev bool
		i.tokenizer.Tokenize(property, func(token Token, original string, _ int, last bool) bool {
			if j == 0 {
				i.Trie.Insert(token, original, uint32(id))
				// Record bigram transitions from the same field that feeds the Trie
				if hasPrev {
					i.Trie.AddTransition(prev, token)
				}
				prev = token
				hasPrev = true
			}
			if l, ok := i.TokenMap[token]; !ok {
				l := roaring.New()
				l.Add(uint32(id))
				i.TokenMap[token] = l
			} else {
				l.Add(uint32(id))
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

			// todo check is clone needed
			return h.All
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
	bm := res.Bitmap()

	mappings := types.CurrentSettings.WordMappings

	i.tokenizer.Tokenize(query, func(token Token, original string, count int, last bool) bool {
		log.Printf("filter on token %s", token)
		ids, found := i.TokenMap[token]
		if found && ids != nil {
			if !ids.IsEmpty() {
				bm.And(ids)
			} else {
				found = false
			}
		}
		if word, ok := mappings[string(token)]; ok {
			ids, found = i.TokenMap[Token(word)]
			if found && ids != nil {
				bm.And(ids)
				found = true
			}

			//}
		}

		if !found {
			tries := roaring.New()
			for _, match := range i.Trie.FindMatches(token) {
				if match.Items != nil && types.HasIntersection(match.Items, bm) {
					tries.Or(match.Items)
				}
			}

			// fuzzy
			fuzzyMatches := i.getBestFuzzyMatch(token, 3)
			for _, match := range fuzzyMatches {
				if fuzzyIds, ok := i.TokenMap[match]; ok && fuzzyIds != nil && types.HasIntersection(fuzzyIds, bm) {
					tries.Or(fuzzyIds)
				}
			}
			bm.And(tries)
		}

		return !bm.IsEmpty()
	})
}

func (i *FreeTextItemHandler) Search(query string) *types.ItemList {
	res := roaring.New()
	//mergeLimit := types.CurrentSettings.SearchMergeLimit
	i.mu.RLock()
	defer i.mu.RUnlock()

	mappings := types.CurrentSettings.WordMappings

	i.tokenizer.Tokenize(query, func(token Token, original string, count int, last bool) bool {
		ids, found := i.TokenMap[token]
		if found {
			if count == 0 {
				res.Or(ids)
			} else if types.HasIntersection(ids, res) {
				res.And(ids)
			} else {
				found = false
			}
		}
		if word, ok := mappings[string(token)]; ok {
			ids, found = i.TokenMap[Token(word)]
			if !found || count == 0 {
				res.Or(ids)
			} else if types.HasIntersection(ids, res) {
				res.And(ids)
			} else {
				found = false
			}
		}
		foundTrie := found
		//log.Printf("word: %s, found: %v, last: %v", token, found, last)
		if !found {
			for _, match := range i.Trie.FindMatches(token) {
				foundTrie = true
				res.And(match.Items)
			}
		}
		if !foundTrie {
			// fuzzy
			fuzzyMatches := i.getBestFuzzyMatch(token, 3)
			for _, match := range fuzzyMatches {
				if fuzzyIds, ok := i.TokenMap[match]; ok {
					res.Or(fuzzyIds)
				}
			}
		}

		return !res.IsEmpty()
	})

	return types.FromBitmap(res)

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
