package search

import "strings"

type Token string

type Tokenizer struct {
	MaxTokens int
}

func simplify(text string) Token {
	return Token(strings.ToLower(text))
}

func (t *Tokenizer) Tokenize(text string) []Token {
	parts := make([]Token, t.MaxTokens)
	c := 0
	lastSplit := 0
	for idx, chr := range text {
		if chr == ' ' || chr == '\n' || chr == '\t' || chr == ',' || chr == ':' || chr == '.' || chr == '!' || chr == '?' || chr == ';' || chr == '(' || chr == ')' || chr == '[' || chr == ']' || chr == '{' || chr == '}' || chr == '"' || chr == '\'' {
			if idx > lastSplit+1 {
				parts[c] = simplify(text[lastSplit:idx])
				c++
				if c >= t.MaxTokens {
					break
				}
			}
			lastSplit = idx + 1
		}
	}
	if lastSplit < len(text) {
		parts[c] = simplify(text[lastSplit:])
		c++
	}

	return parts[:c]
}
