package search

import "testing"

func TestTrie(t *testing.T) {
	trie := NewTrie()
	trie.Insert("hello", 1)
	trie.Insert("world", 1)
	trie.Insert("hell", 2)
	trie.Insert("cat", 2)
	trie.Insert("dog", 3)
	trie.Insert("doggo", 4)
	trie.Insert("doggy", 1)
	trie.Insert("dogger", 2)
	trie.Insert("dogging", 4)
	trie.Insert("dogged", 2)

	if !trie.Search("hello") {
		t.Error("Expected to find hello")
	}
	if !trie.Search("world") {
		t.Error("Expected to find world")
	}
	matching := trie.FindMatches("dog")
	if len(matching) != 6 {
		t.Error("Expected 6 matches for dog")
	}

	matching = trie.FindMatches("he")
	if len(matching) != 2 {
		t.Error("Expected 2 matches for he")
	}
}
