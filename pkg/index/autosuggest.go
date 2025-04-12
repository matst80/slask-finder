package index

// import (
// 	"sync"

// 	"github.com/matst80/slask-finder/pkg/search"
// 	"github.com/matst80/slask-finder/pkg/types"
// )

// type AutoSuggest struct {
// 	mu        sync.RWMutex
// 	tokenizer *search.Tokenizer
// 	Trie      *search.Trie
// }

// func NewAutoSuggest(tokenizer *search.Tokenizer) *AutoSuggest {
// 	return &AutoSuggest{
// 		mu:        sync.RWMutex{},
// 		tokenizer: tokenizer,
// 		Trie:      search.NewTrie(),
// 	}
// }

// func (a *AutoSuggest) Insert(word string, item types.Item) {
// 	a.mu.Lock()
// 	defer a.mu.Unlock()
// 	a.insertUnsafe(word, item)
// }

// func (a *AutoSuggest) insertUnsafe(word string, item types.Item) {
// 	if len(word) > 1 {
// 		a.Trie.Insert(word, item)
// 	}
// }

// func (a *AutoSuggest) InsertItem(item types.Item) {
// 	a.mu.Lock()
// 	defer a.mu.Unlock()

// 	a.InsertItemUnsafe(item)
// }

// func (a *AutoSuggest) Lock() {
// 	a.mu.Lock()
// }

// func (a *AutoSuggest) Unlock() {
// 	a.mu.Unlock()
// }

// func (a *AutoSuggest) InsertItemUnsafe(item types.Item) {
// 	title := item.GetTitle()
// 	search.SplitWords(title, func(word string, count int) bool {
// 		a.insertUnsafe(word, item)
// 		return true
// 	})
// }

// // func (a *AutoSuggest) FindMatches(text string) []search.Match {
// // 	a.mu.RLock()
// // 	defer a.mu.RUnlock()

// // 	return a.Trie.FindMatches(text)
// // }

// func (a *AutoSuggest) FindMatchesForWord(word string, resultChan chan<- []search.Match) {
// 	tokens := a.tokenizer.Tokenize(word)
// 	if len(tokens) == 0 {
// 		resultChan <- []search.Match{}
// 		return
// 	}
// 	resultChan <- a.Trie.FindMatches(tokens[len(tokens)-1])
// }
