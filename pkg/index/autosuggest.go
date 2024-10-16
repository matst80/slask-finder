package index

import (
	"strings"
	"sync"

	"tornberg.me/facet-search/pkg/search"
)

type AutoSuggest struct {
	mu   sync.RWMutex
	Trie *search.Trie
}

func (a *AutoSuggest) Insert(word string, id uint) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.insertUnsafe(word, id)
}

func (a *AutoSuggest) insertUnsafe(word string, id uint) {
	if len(word) > 1 {
		a.Trie.Insert(word, id)
	}
}

func (a *AutoSuggest) InsertItem(item *DataItem) {
	a.mu.Lock()
	defer a.mu.Unlock()

	addItem := func(word string, count int) bool {
		a.insertUnsafe(word, item.Id)
		return true
	}
	search.SplitWords(strings.ToLower(item.Title), addItem)
}

func (a *AutoSuggest) FindMatches(text string) []search.Match {
	a.mu.RLock()
	defer a.mu.RUnlock()
	words := strings.Split(strings.ToLower(text), " ")
	// for i, word := range words[:len(words)-1] {
	// 	a.Trie.FindMatches(strings.ToLower(word))
	// }
	prefix := words[len(words)-1]
	return a.Trie.FindMatches(strings.ToLower(prefix))
}

func (a *AutoSuggest) FindMatchesForWord(word string, resultChan chan<- []search.Match) {
	resultChan <- a.Trie.FindMatches(strings.ToLower(word))
}
