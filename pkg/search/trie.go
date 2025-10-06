package search

import (
	"sort"

	"github.com/RoaringBitmap/roaring/v2"
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
	Items    *roaring.Bitmap
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

// untested
func (t *Trie) RemoveDocument(id uint32) {
	var removeHelper func(node *Node) bool
	removeHelper = func(node *Node) bool {
		if node.Items != nil {
			node.Items.Remove(id)
		}
		// Recursively clean up children
		for r, child := range node.Children {
			if removeHelper(child) {
				delete(node.Children, r)
			}
		}
		// If this node has no items and no children, it can be removed
		return node.Items.GetCardinality() == 0 && len(node.Children) == 0
	}
	removeHelper(t.Root)
}

func (t *Trie) Insert(word Token, raw string, id uint32) {
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
		node.Items = roaring.New()
	}
	node.Items.Add(id)

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
				li = int(matches[i].Items.Bitmap().GetCardinality())
				// li = len(*matches[i].Items)
			}
			lj := 0
			if matches[j].Items != nil {
				lj = int(matches[j].Items.Bitmap().GetCardinality())
				// lj = len(*matches[j].Items)
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
				li = int(scoredMatches[i].m.Items.Bitmap().GetCardinality())
				//li = len(*scoredMatches[i].m.Items)
			}
			lj := 0
			if scoredMatches[j].m.Items != nil {
				lj = int(scoredMatches[j].m.Items.Bitmap().GetCardinality())
				//lj = len(*scoredMatches[j].m.Items)
			}
			return li > lj
		})
	} else {
		// All counts are zero -> fallback by popularity
		sort.SliceStable(scoredMatches, func(i, j int) bool {
			li := 0
			if scoredMatches[i].m.Items != nil {
				li = int(scoredMatches[i].m.Items.Bitmap().GetCardinality())
				//li = len(*scoredMatches[i].m.Items)
			}
			lj := 0
			if scoredMatches[j].m.Items != nil {
				lj = int(scoredMatches[j].m.Items.Bitmap().GetCardinality())
				//lj = len(*scoredMatches[j].m.Items)
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
			Items:  types.FromBitmap(node.Items),
		})
	}
	for r, child := range node.Children {
		matches = append(matches, t.findMatches(child, prefix+string(r))...)
	}
	return matches
}

// PredictSequence completes the first word from the given prefix using the
// previous token context, then greedily predicts subsequent words by following
// the highest-count Markov transitions until no transition exists, a loop is
// detected, or maxWords is reached. Returns the sequence as display words.
func (t *Trie) PredictSequence(prev Token, prefix Token, maxWords int) []string {
	if maxWords <= 0 {
		maxWords = 1
	}
	matches := t.FindMatchesWithPrev(prefix, prev)
	if len(matches) == 0 {
		return []string{}
	}
	seqTokens := make([]Token, 0, maxWords)
	firstToken := Token(matches[0].Prefix)
	seqTokens = append(seqTokens, firstToken)

	visited := map[Token]struct{}{firstToken: {}}
	current := firstToken
	for len(seqTokens) < maxWords {
		nextMap, ok := t.transitions[current]
		if !ok || len(nextMap) == 0 {
			break
		}
		// pick highest-count next; tiebreak by Items popularity if both are words in trie
		var best Token
		bestCount := -1
		bestPop := -1
		for cand, cnt := range nextMap {
			if _, seen := visited[cand]; seen {
				continue // avoid loops
			}
			pop := 0
			if n := t.Search(string(cand)); n != nil && n.IsLeaf && n.Items != nil {
				pop = int(n.Items.GetCardinality()) // len(n.Items)
			}
			if cnt > bestCount || (cnt == bestCount && pop > bestPop) {
				best = cand
				bestCount = cnt
				bestPop = pop
			}
		}
		if bestCount <= 0 {
			break
		}
		seqTokens = append(seqTokens, best)
		visited[best] = struct{}{}
		current = best
	}

	// map tokens back to display words (raw), fallback to normalized if not found
	result := make([]string, 0, len(seqTokens))
	for _, tok := range seqTokens {
		if n := t.Search(string(tok)); n != nil && n.IsLeaf && len(n.Word) > 0 {
			result = append(result, n.Word)
		} else {
			result = append(result, string(tok))
		}
	}
	return result
}

type PredictionNode struct {
	Word     string           `json:"word"`
	Count    int              `json:"count,omitempty"`
	Children []PredictionNode `json:"children,omitempty"`
}

// PredictTree builds a tree of depth maxDepth using top-k Markov transitions.
// The first level is chosen from trie matches by FindMatchesWithPrev on the given prefix.
func (t *Trie) PredictTree(prev Token, prefix Token, maxDepth int, k int) []PredictionNode {
	if maxDepth <= 0 {
		return nil
	}
	if k <= 0 {
		k = 3
	}
	matches := t.FindMatchesWithPrev(prefix, prev)
	if len(matches) == 0 {
		return nil
	}
	limit := min(len(matches), k)
	nodes := make([]PredictionNode, 0, limit)
	for i := range limit {
		m := matches[i]
		tok := Token(m.Prefix)
		count := 0
		if tr, ok := t.transitions[prev]; ok {
			count = tr[tok]
		}
		node := PredictionNode{Word: m.Word, Count: count}
		node.Children = t.predictChildren(tok, maxDepth-1, k, map[Token]struct{}{tok: {}})
		nodes = append(nodes, node)
	}
	return nodes
}

func (t *Trie) predictChildren(current Token, remainingDepth int, k int, visited map[Token]struct{}) []PredictionNode {
	if remainingDepth <= 0 {
		return nil
	}
	nextMap, ok := t.transitions[current]
	if !ok || len(nextMap) == 0 {
		return nil
	}
	// collect candidates
	type cand struct {
		tok Token
		cnt int
		pop uint64
	}
	cands := make([]cand, 0, len(nextMap))
	for tok, cnt := range nextMap {
		if _, seen := visited[tok]; seen {
			continue
		}
		pop := uint64(0)
		if n := t.Search(string(tok)); n != nil && n.IsLeaf && n.Items != nil {
			pop = n.Items.GetCardinality()
		}
		cands = append(cands, cand{tok: tok, cnt: cnt, pop: pop})
	}
	if len(cands) == 0 {
		return nil
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].cnt != cands[j].cnt {
			return cands[i].cnt > cands[j].cnt
		}
		return cands[i].pop > cands[j].pop
	})
	limit := min(len(cands), k)
	res := make([]PredictionNode, 0, limit)
	for i := range limit {
		c := cands[i]
		word := string(c.tok)
		if n := t.Search(word); n != nil && n.IsLeaf && len(n.Word) > 0 {
			word = n.Word
		}
		node := PredictionNode{Word: word, Count: c.cnt}
		// extend visited path
		path := make(map[Token]struct{}, len(visited)+1)
		for k := range visited {
			path[k] = struct{}{}
		}
		path[c.tok] = struct{}{}
		node.Children = t.predictChildren(c.tok, remainingDepth-1, k, path)
		res = append(res, node)
	}
	return res
}
