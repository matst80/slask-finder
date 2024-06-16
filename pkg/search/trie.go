package search

type Trie struct {
	Root *Node
}

type Node struct {
	Children map[rune]*Node
	IsLeaf   bool
}

func NewTrie() *Trie {
	return &Trie{
		Root: &Node{
			Children: make(map[rune]*Node),
		},
	}
}

func (t *Trie) Insert(word string) {
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

func (t *Trie) FindMatches(prefix string) []string {
	node := t.Root
	for _, r := range prefix {
		if _, ok := node.Children[r]; !ok {
			return nil
		}
		node = node.Children[r]
	}
	return t.findMatches(node, prefix)
}

func (t *Trie) findMatches(node *Node, prefix string) []string {
	var matches []string
	if node.IsLeaf {
		matches = append(matches, prefix)
	}
	for r, child := range node.Children {
		matches = append(matches, t.findMatches(child, prefix+string(r))...)
	}
	return matches
}
