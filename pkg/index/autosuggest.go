package index

import (
	"strings"
	"sync"

	"tornberg.me/facet-search/pkg/search"
	"tornberg.me/facet-search/pkg/types"
)

type AutoSuggest struct {
	mu   sync.RWMutex
	Trie *search.Trie
}

func (a *AutoSuggest) Insert(word string, item types.Item) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.insertUnsafe(word, item)
}

func (a *AutoSuggest) insertUnsafe(word string, item types.Item) {
	if len(word) > 1 {
		a.Trie.Insert(word, item)
	}
}

func (a *AutoSuggest) InsertItem(item types.Item) {
	a.mu.Lock()
	defer a.mu.Unlock()

	addItem := func(word string, count int) bool {
		a.insertUnsafe(word, item)
		return true
	}
	title := strings.ToLower(item.GetTitle())
	search.SplitWords(strings.ToLower(title), addItem)
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
