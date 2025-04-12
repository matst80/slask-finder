package index

import (
	"sync"

	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
)

type AutoSuggest struct {
	mu        sync.RWMutex
	tokenizer *search.Tokenizer
	Trie      *search.Trie
}

func NewAutoSuggest(tokenizer *search.Tokenizer) *AutoSuggest {
	return &AutoSuggest{
		mu:        sync.RWMutex{},
		tokenizer: tokenizer,
		Trie:      search.NewTrie(),
	}
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

	// addItem := func(word string, count int) bool {
	// 	a.insertUnsafe(word, item)
	// 	return true
	// }
	title := item.GetTitle()
	search.SplitWords(title, func(word string, count int) bool {
		a.insertUnsafe(word, item)
		return true
	})
	// for _, word := range a.tokenizer.Tokenize(title) {
	// 	a.insertUnsafe(string(word), item)
	// }
	//search.SplitWords(strings.ToLower(title), addItem)
}

func (a *AutoSuggest) FindMatches(text string) []search.Match {
	a.mu.RLock()
	defer a.mu.RUnlock()
	words := a.tokenizer.Tokenize(text)
	//words := strings.Split(strings.ToLower(text), " ")
	// for i, word := range words[:len(words)-1] {
	// 	a.Trie.FindMatches(strings.ToLower(word))
	// }
	suffix := words[len(words)-1]
	return a.Trie.FindMatches(string(suffix))
}

func (a *AutoSuggest) FindMatchesForWord(word string, resultChan chan<- []search.Match) {
	tokens := a.tokenizer.Tokenize(word)
	resultChan <- a.Trie.FindMatches(string(tokens[len(tokens)-1]))
}
