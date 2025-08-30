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

func TestTrie_PredictSequence(t *testing.T) {
	trie := NewTrie()
	// Insert a small vocabulary
	trie.Insert(Token("hello"), "hello", 1)
	trie.Insert(Token("world"), "world", 2)
	trie.Insert(Token("iphone"), "iPhone", 3)
	trie.Insert(Token("ipad"), "iPad", 4)
	trie.Insert(Token("pro"), "Pro", 5)

	// Build transitions: apple -> iphone (3), iphone -> pro (2), apple -> ipad (1)
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("ipad"))
	trie.AddTransition(Token("iphone"), Token("pro"))
	trie.AddTransition(Token("iphone"), Token("pro"))

	seq := trie.PredictSequence(Token("apple"), Token("ip"), 3)
	// Expect termination when no transition from "pro": length 2 (iPhone Pro)
	if len(seq) != 2 {
		t.Fatalf("expected sequence length 2, got %d (%v)", len(seq), seq)
	}
	if seq[0] != "iPhone" || seq[1] != "Pro" {
		t.Fatalf("expected beginning 'iPhone Pro', got %v", seq)
	}

	seq2 := trie.PredictSequence(Token("world"), Token("he"), 2)
	if len(seq2) != 1 || seq2[0] != "hello" {
		t.Fatalf("expected only 'hello' when no transitions from 'world', got %v", seq2)
	}
}
