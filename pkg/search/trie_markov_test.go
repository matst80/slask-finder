package search

import (
	"testing"
)

func TestTrie_MarkovRankingAndFallback(t *testing.T) {
	trie := NewTrie()

	// Insert candidates under prefix "ip"
	trie.Insert(Token("iphone"), "iPhone", 1) // 1 item
	trie.Insert(Token("ipad"), "iPad", 2)     // 2 items
	trie.Insert(Token("ipad"), "iPad", 3)

	// Sanity: matches exist for prefix
	matches := trie.FindMatches(Token("ip"))
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for prefix 'ip', got %d", len(matches))
	}

	// Fallback ranking (no transitions for prev="samsung"): should rank by item popularity
	fallback := trie.FindMatchesWithPrev(Token("ip"), Token("samsung"))
	if len(fallback) != 2 {
		t.Fatalf("expected 2 fallback matches, got %d", len(fallback))
	}
	if fallback[0].Word != "iPad" {
		t.Fatalf("expected fallback top match to be iPad (more items), got %s", fallback[0].Word)
	}

	// Add Markov transitions favoring "iphone" after prev="apple"
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("ipad")) // fewer transitions

	ranked := trie.FindMatchesWithPrev(Token("ip"), Token("apple"))
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked matches, got %d", len(ranked))
	}
	if ranked[0].Word != "iPhone" {
		t.Fatalf("expected Markov-ranked top match to be iPhone, got %s", ranked[0].Word)
	}
}
