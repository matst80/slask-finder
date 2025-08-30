package search

import (
	"math"
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

// Tunable weights for attention-based scoring in sequence prediction
const (
	attnWLocal = 0.6  // weight for local continuity P(next|current)
	attnWFirst = 0.35 // weight for first-word attention P(next|first)
	attnWPop   = 0.05 // small weight for popularity prior
	attnDecay  = 0.85 // decay per step for first-word attention
	attnAlpha  = 0.5  // add-one style smoothing (alpha)
	beamWidth  = 5    // beam width for PredictSequence
	maxBranch  = 20   // cap next candidates per expansion to avoid blowup
)

func (t *Trie) probNext(prev, next Token) float64 {
	// Smoothed probability (alpha-smoothing)
	if prev == "" || next == "" {
		return 1e-12
	}
	nextMap, ok := t.transitions[prev]
	if !ok || len(nextMap) == 0 {
		return 1e-12
	}
	V := float64(len(nextMap))
	count := float64(nextMap[next])
	tot := float64(t.totals[prev])
	return (count + attnAlpha) / (tot + attnAlpha*V)
}

func (t *Trie) popPrior(next Token) float64 {
	// Popularity prior via items count on token
	if n := t.Search(string(next)); n != nil && n.IsLeaf {
		return float64(len(n.Items)) + 1.0 // +1 to avoid log(0)
	}
	return 1.0
}

// PredictSequence completes the first word from the given prefix using the
// previous token context, then predicts subsequent words with a beam search
// that mixes local continuity and first-word attention, plus a tiny popularity prior.
func (t *Trie) PredictSequence(prev Token, prefix Token, maxWords int) []string {
	if maxWords <= 0 {
		maxWords = 1
	}
	matches := t.FindMatchesWithPrev(prefix, prev)
	if len(matches) == 0 {
		return []string{}
	}
	first := Token(matches[0].Prefix)

	type state struct {
		last    Token
		first   Token
		path    []Token
		score   float64
		visited map[Token]struct{}
	}

	beam := make([]state, 1)
	beam[0] = state{
		last:    first,
		first:   first,
		path:    []Token{first},
		score:   0,
		visited: map[Token]struct{}{first: {}},
	}

	step := 1
	for len(beam) > 0 && len(beam[0].path) < maxWords {
		candidates := make([]state, 0, beamWidth*5)
		for _, s := range beam {
			nextMap, ok := t.transitions[s.last]
			if !ok || len(nextMap) == 0 {
				continue
			}
			// collect and sort next options by local count to prune
			type opt struct {
				tok Token
				cnt int
			}
			opts := make([]opt, 0, len(nextMap))
			for tok, cnt := range nextMap {
				if _, seen := s.visited[tok]; !seen {
					opts = append(opts, opt{tok, cnt})
				}
			}
			sort.SliceStable(opts, func(i, j int) bool { return opts[i].cnt > opts[j].cnt })
			limit := len(opts)
			if limit > maxBranch {
				limit = maxBranch
			}
			for i := 0; i < limit; i++ {
				nextTok := opts[i].tok
				pLocal := t.probNext(s.last, nextTok)
				pFirst := t.probNext(s.first, nextTok)
				pPop := t.popPrior(nextTok)
				// mix in log-space
				stepWeight := math.Pow(attnDecay, float64(step-1))
				sc := s.score + attnWLocal*math.Log(pLocal) + attnWFirst*stepWeight*math.Log(pFirst) + attnWPop*math.Log(pPop)
				// new state
				visited := make(map[Token]struct{}, len(s.visited)+1)
				for k := range s.visited {
					visited[k] = struct{}{}
				}
				visited[nextTok] = struct{}{}
				path := append(append([]Token{}, s.path...), nextTok)
				candidates = append(candidates, state{last: nextTok, first: s.first, path: path, score: sc, visited: visited})
			}
		}
		if len(candidates) == 0 {
			break
		}
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
		if len(candidates) > beamWidth {
			candidates = candidates[:beamWidth]
		}
		beam = candidates
		step++
	}

	best := beam[0]
	// map tokens back to display words (raw), fallback to normalized if not found
	result := make([]string, 0, len(best.path))
	for _, tok := range best.path {
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
	limit := k
	if len(matches) < limit {
		limit = len(matches)
	}
	nodes := make([]PredictionNode, 0, limit)
	for i := 0; i < limit; i++ {
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
		pop int
	}
	cands := make([]cand, 0, len(nextMap))
	for tok, cnt := range nextMap {
		if _, seen := visited[tok]; seen {
			continue
		}
		pop := 0
		if n := t.Search(string(tok)); n != nil && n.IsLeaf && n.Items != nil {
			pop = len(n.Items)
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
	limit := k
	if len(cands) < limit {
		limit = len(cands)
	}
	res := make([]PredictionNode, 0, limit)
	for i := 0; i < limit; i++ {
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
