package index

import (
	"strings"

	"tornberg.me/facet-search/pkg/search"
)

type AutoSuggest struct {
	Trie *search.Trie
}

func (a *AutoSuggest) Insert(word string, id uint) {
	if len(word) > 1 {
		a.Trie.Insert(word, id)
	}
}

func (a *AutoSuggest) InsertItem(item *DataItem) {
	for _, word := range strings.Split(strings.ToLower(item.Title), " ") {
		if len(word) > 1 {
			a.Trie.Insert(word, item.Id)
		}
	}
}

func (a *AutoSuggest) FindMatches(text string) []search.Match {
	words := strings.Split(strings.ToLower(text), " ")
	// for i, word := range words[:len(words)-1] {
	// 	a.Trie.FindMatches(strings.ToLower(word))
	// }
	prefix := words[len(words)-1]
	return a.Trie.FindMatches(strings.ToLower(prefix))
}
