package search

// import (
// 	"testing"

// 	"github.com/matst80/slask-finder/pkg/types"
// )

// func mockItem(id uint) types.Item {
// 	return &types.MockItem{
// 		Id: id,
// 	}

// }

// func TestTrie(t *testing.T) {
// 	trie := NewTrie()
// 	trie.Insert("hello", mockItem(1))
// 	trie.Insert("world", mockItem(1))
// 	trie.Insert("hell", mockItem(2))
// 	trie.Insert("cat", mockItem(2))
// 	trie.Insert("dog", mockItem(3))
// 	trie.Insert("doggo", mockItem(4))
// 	trie.Insert("doggy", mockItem(1))
// 	trie.Insert("dogger", mockItem(2))
// 	trie.Insert("dogging", mockItem(4))
// 	trie.Insert("dogged", mockItem(2))

// 	if !trie.Search("hello") {
// 		t.Error("Expected to find hello")
// 	}
// 	if !trie.Search("world") {
// 		t.Error("Expected to find world")
// 	}
// 	matching := trie.FindMatches("dog")
// 	if len(matching) != 6 {
// 		t.Error("Expected 6 matches for dog")
// 	}

// 	matching = trie.FindMatches("he")
// 	if len(matching) != 2 {
// 		t.Error("Expected 2 matches for he")
// 	}
// }
