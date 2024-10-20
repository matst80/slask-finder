package search

import "tornberg.me/facet-search/pkg/types"

type Trie struct {
	Root *Node
}

type Node struct {
	Children map[rune]*Node
	IsLeaf   bool
	Items    types.ItemList
}

func NewTrie() *Trie {
	return &Trie{
		Root: &Node{
			Children: make(map[rune]*Node),
		},
	}
}

func (t *Trie) Insert(word string, item types.Item) {
	node := t.Root
	for _, r := range word {
		if _, ok := node.Children[r]; !ok {
			node.Children[r] = &Node{
				Children: make(map[rune]*Node),
			}
		}
		node = node.Children[r]
	}
	id := item.GetId()
	node.IsLeaf = true
	if node.Items == nil {
		node.Items = types.ItemList{id: item}
	}
	node.Items[id] = item
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
	Word  string          `json:"word"`
	Items *types.ItemList `json:"ids"`
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
			Word:  prefix,
			Items: &node.Items,
		})
	}
	for r, child := range node.Children {
		matches = append(matches, t.findMatches(child, prefix+string(r))...)
	}
	return matches
}
