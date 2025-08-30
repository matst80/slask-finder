package search

import (
	"sort"

	"github.com/matst80/slask-finder/pkg/types"
)

type Trie struct {
	Root *Node
	// Markov bigram transitions: prev token -> next token -> count
	transitions map[Token]map[Token]int
	// Totals per prev token for normalization/backoff if needed
	totals map[Token]int
}

type Node struct {
	Children map[rune]*Node
	IsLeaf   bool
	Word     string
	Items    types.ItemList
}

func NewTrie() *Trie {
	return &Trie{
		Root: &Node{
			Children: make(map[rune]*Node),
		},
		transitions: make(map[Token]map[Token]int),
		totals:      make(map[Token]int),
	}
}

func (t *Trie) Insert(word Token, raw string, id uint) {
	node := t.Root

	for _, r := range word {
		if _, ok := node.Children[r]; !ok {
			node.Children[r] = &Node{
				Children: make(map[rune]*Node),
			}
		}
		node = node.Children[r]
	}

	node.IsLeaf = true
	node.Word = raw
	if node.Items == nil {
		node.Items = types.ItemList{id: struct{}{}}
	} else {
		node.Items.AddId(id)
	}
}

// AddTransition increments the bigram count from prev -> next.
func (t *Trie) AddTransition(prev, next Token) {
	if len(prev) == 0 || len(next) == 0 {
		return
	}
	m, ok := t.transitions[prev]
	if !ok {
		m = make(map[Token]int)
		t.transitions[prev] = m
	}
	m[next]++
	t.totals[prev]++
}

func (t *Trie) Search(word string) *Node {
	node := t.Root
	for _, r := range word {
		if _, ok := node.Children[r]; !ok {
			return nil
		}
		node = node.Children[r]
	}
	if node.IsLeaf {
		return node
	}
	return nil
}

type Match struct {
	Prefix string          `json:"prefix"`
	Word   string          `json:"word"`
	Items  *types.ItemList `json:"ids"`
}

func (t *Trie) FindMatches(prefix Token) []Match {
	node := t.Root
	for _, r := range prefix {
		if _, ok := node.Children[r]; !ok {
			return nil
		}
		node = node.Children[r]
	}
	return t.findMatches(node, string(prefix))
}

// FindMatchesWithPrev returns matches for the given prefix, ranked by the
// Markov transition counts from the provided previous token. Falls back to
// Item frequency when no transition data exists.
func (t *Trie) FindMatchesWithPrev(prefix Token, prev Token) []Match {
	matches := t.FindMatches(prefix)
	if len(matches) <= 1 {
		return matches
	}
	trans, hasTrans := t.transitions[prev]
	if !hasTrans {
		// No transitions at all for this prev -> fallback by popularity
		sort.SliceStable(matches, func(i, j int) bool {
			li := 0
			if matches[i].Items != nil {
				li = len(*matches[i].Items)
			}
			lj := 0
			if matches[j].Items != nil {
				lj = len(*matches[j].Items)
			}
			return li > lj
		})
		return matches
	}

	// Compute counts and determine if any positive counts exist
	type scored struct {
		m Match
		c int
	}
	scoredMatches := make([]scored, 0, len(matches))
	anyPositive := false
	for _, m := range matches {
		c := trans[Token(m.Prefix)]
		if c > 0 {
			anyPositive = true
		}
		scoredMatches = append(scoredMatches, scored{m: m, c: c})
	}

	if anyPositive {
		// Sort primarily by transition count desc, tiebreak by popularity
		sort.SliceStable(scoredMatches, func(i, j int) bool {
			if scoredMatches[i].c != scoredMatches[j].c {
				return scoredMatches[i].c > scoredMatches[j].c
			}
			li := 0
			if scoredMatches[i].m.Items != nil {
				li = len(*scoredMatches[i].m.Items)
			}
			lj := 0
			if scoredMatches[j].m.Items != nil {
				lj = len(*scoredMatches[j].m.Items)
			}
			return li > lj
		})
	} else {
		// All counts are zero -> fallback by popularity
		sort.SliceStable(scoredMatches, func(i, j int) bool {
			li := 0
			if scoredMatches[i].m.Items != nil {
				li = len(*scoredMatches[i].m.Items)
			}
			lj := 0
			if scoredMatches[j].m.Items != nil {
				lj = len(*scoredMatches[j].m.Items)
			}
			return li > lj
		})
	}

	// Unwrap
	for i := range scoredMatches {
		matches[i] = scoredMatches[i].m
	}
	return matches
}

func (t *Trie) findMatches(node *Node, prefix string) []Match {
	var matches []Match
	if node.IsLeaf {
		matches = append(matches, Match{
			Prefix: prefix,
			Word:   node.Word,
			Items:  &node.Items,
		})
	}
	for r, child := range node.Children {
		matches = append(matches, t.findMatches(child, prefix+string(r))...)
	}
	return matches
}
