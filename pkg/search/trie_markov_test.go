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

func TestTrie_PredictSequence_WithAttention(t *testing.T) {
	trie := NewTrie()
	// Vocabulary
	trie.Insert(Token("iphone"), "iPhone", 1)
	trie.Insert(Token("ipad"), "iPad", 2)
	trie.Insert(Token("pro"), "Pro", 3)
	trie.Insert(Token("case"), "case", 4)
	trie.Insert(Token("charger"), "charger", 5)

	// Transitions
	// Strongly favor: apple -> iphone; iphone -> pro
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("iphone"), Token("pro"))
	trie.AddTransition(Token("iphone"), Token("pro"))

	// Distractor: after pro, local bigram prefers accessories (case, charger)
	trie.AddTransition(Token("pro"), Token("case"))
	trie.AddTransition(Token("pro"), Token("case"))
	trie.AddTransition(Token("pro"), Token("charger"))

	// Competing path if attention is weak: apple -> ipad (1) then accessories
	trie.AddTransition(Token("apple"), Token("ipad"))
	trie.AddTransition(Token("ipad"), Token("case"))
	trie.AddTransition(Token("ipad"), Token("charger"))

	// Predict with prefix "ip" and prev "apple". With attention to first word,
	// sequence should start with iPhone and keep coherent next as Pro before accessories.
	seq := trie.PredictSequence(Token("apple"), Token("ip"), 3)
	if len(seq) < 2 {
		t.Fatalf("expected at least 2 tokens, got %v", seq)
	}
	if seq[0] != "iPhone" {
		t.Fatalf("expected first token iPhone, got %v", seq)
	}
	if seq[1] != "Pro" {
		t.Fatalf("expected second token Pro influenced by attention, got %v", seq)
	}
}

func TestTrie_PredictTree_Attention(t *testing.T) {
	trie := NewTrie()
	// Vocabulary
	trie.Insert(Token("iphone"), "iPhone", 1)
	trie.Insert(Token("ipad"), "iPad", 2)
	trie.Insert(Token("pro"), "Pro", 3)
	trie.Insert(Token("air"), "Air", 4)
	trie.Insert(Token("case"), "case", 5)
	trie.Insert(Token("charger"), "charger", 6)

	// Transitions
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("iphone"))
	trie.AddTransition(Token("apple"), Token("ipad"))
	trie.AddTransition(Token("iphone"), Token("pro"))
	trie.AddTransition(Token("iphone"), Token("air"))
	trie.AddTransition(Token("pro"), Token("case"))
	trie.AddTransition(Token("pro"), Token("charger"))
	trie.AddTransition(Token("ipad"), Token("case"))

	tree := trie.PredictTree(Token("apple"), Token("ip"), 2, 3)
	if len(tree) == 0 {
		t.Fatalf("expected non-empty tree")
	}
	if tree[0].Word != "iPhone" { // first level should prefer iPhone over iPad due to transitions
		t.Fatalf("expected first node iPhone, got %v", tree[0].Word)
	}
	if len(tree[0].Children) == 0 {
		t.Fatalf("expected children under first node")
	}
	// children should prioritize Pro then Air (attention + local), not accessories directly
	if tree[0].Children[0].Word != "Pro" && tree[0].Children[0].Word != "Air" {
		t.Fatalf("expected Pro/Air as first child, got %v", tree[0].Children[0].Word)
	}
}
