package search

import "testing"

func TestTrie(t *testing.T) {
	trie := NewTrie()
	trie.Insert("hello")
	trie.Insert("world")
	trie.Insert("hell")
	trie.Insert("cat")
	trie.Insert("dog")
	trie.Insert("doggo")
	trie.Insert("doggy")
	trie.Insert("dogger")
	trie.Insert("dogging")
	trie.Insert("dogged")

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
