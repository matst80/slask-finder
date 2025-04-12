package search

import "github.com/matst80/slask-finder/pkg/types"

type Trie struct {
	Root *Node
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
	}
}

func (t *Trie) Insert(word string, item types.Item) {
	node := t.Root

	for _, r := range NormalizeWord(word) {
		if _, ok := node.Children[r]; !ok {
			node.Children[r] = &Node{
				Children: make(map[rune]*Node),
			}
		}
		node = node.Children[r]
	}
	id := item.GetId()
	node.IsLeaf = true
	node.Word = word
	if node.Items == nil {
		node.Items = types.ItemList{id: struct{}{}}
	} else {
		node.Items.AddId(id)
	}
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
