package search

import "tornberg.me/facet-search/pkg/facet"

type Trie struct {
	Root *Node
}

type Node struct {
	Children map[rune]*Node
	IsLeaf   bool
	Items    facet.IdList
}

func NewTrie() *Trie {
	return &Trie{
		Root: &Node{
			Children: make(map[rune]*Node),
		},
	}
}

func (t *Trie) Insert(word string, id uint) {
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
	if node.Items == nil {
		node.Items = facet.IdList{id: struct{}{}}
	}
	node.Items[id] = struct{}{}
}

func (t *Trie) Search(word string) bool {
	node := t.Root
	for _, r := range word {
		if _, ok := node.Children[r]; !ok {
			return false
		}
		node = node.Children[r]
	}
	return node.IsLeaf
}

type Match struct {
	Word string        `json:"word"`
	Ids  *facet.IdList `json:"ids"`
}

func (t *Trie) FindMatches(prefix string) []Match {
	node := t.Root
	for _, r := range prefix {
		if _, ok := node.Children[r]; !ok {
			return nil
		}
		node = node.Children[r]
	}
	return t.findMatches(node, prefix)
}

func (t *Trie) findMatches(node *Node, prefix string) []Match {
	var matches []Match
	if node.IsLeaf {
		matches = append(matches, Match{
			Word: prefix,
			Ids:  &node.Items,
		})
	}
	for r, child := range node.Children {
		matches = append(matches, t.findMatches(child, prefix+string(r))...)
	}
	return matches
}
