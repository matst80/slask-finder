package search

import "strings"

type Token string

type Tokenizer struct {
	MaxTokens int
}

type CharReplacement struct {
	From string
	To   string
}

type TokenList []Token

func (t *TokenList) AddToken(token Token) {
	for _, existing := range *t {
		if existing == token {
			return
		}
	}
	*t = append(*t, token)
}

var replacements = []CharReplacement{
	{From: "mobil", To: "smartphone"},
	{From: "ö", To: "o"},
	{From: "ä", To: "a"},
	{From: "å", To: "a"},
	{From: "é", To: "e"},
	{From: "è", To: "e"},
	{From: "ê", To: "e"},
	{From: "ë", To: "e"},
	{From: "ï", To: "i"},
	{From: "î", To: "i"},
	{From: "ö", To: "o"},
	{From: "ô", To: "o"},
	{From: "ü", To: "u"},
	{From: "û", To: "u"},
	{From: "ÿ", To: "y"},
	{From: "ç", To: "c"},
	{From: "ñ", To: "n"},
	{From: "ß", To: "s"},
	{From: "æ", To: "a"},
	{From: "ø", To: "o"},
}

func replaceCommonIssues(text string) string {
	for _, replacement := range replacements {
		text = strings.ReplaceAll(text, replacement.From, replacement.To)
	}
	return text
}

func simplify(text string) Token {

	return Token(replaceCommonIssues(strings.ToLower(text)))
}

func getUniqueTokens(tokens []Token) []Token {
	unique := make(map[Token]bool)
	for _, token := range tokens {
		unique[token] = true
	}
	res := make([]Token, len(unique))
	c := 0
	for token := range unique {
		if len(token) > 0 {
			res[c] = token
			c++
		}
	}
	return res[:c]
}

func SplitWords(text string, onWord func(word string, count int) bool) {
	count := 0
	lastSplit := 0
	for idx, chr := range text {
		if chr == ' ' || chr == '\n' || chr == '\t' || chr == ',' || chr == ':' || chr == '.' || chr == '!' || chr == '?' || chr == ';' || chr == '(' || chr == ')' || chr == '[' || chr == ']' || chr == '{' || chr == '}' || chr == '"' || chr == '\'' || chr == '/' {
			if idx > lastSplit+1 {
				if !onWord(text[lastSplit:idx], count) {
					return
				}
				count++

			}
			lastSplit = idx + 1
		}
	}
	if lastSplit < len(text) {
		onWord(text[lastSplit:], count)
	}
}

func (t *Tokenizer) Tokenize(text string) []Token {
	parts := []Token{}
	c := 0
	SplitWords(text, func(word string, count int) bool {
		if c >= t.MaxTokens {
			return false
		}
		parts = append(parts, simplify(word))
		c++
		return true
	})

	return getUniqueTokens(parts[:c])
}

func (t *Tokenizer) MakeDocument(id uint, text ...string) *Document {

	tokens := []Token{}
	for _, txt := range text {
		tokens = append(tokens, t.Tokenize(txt)...)
	}
	return &Document{
		Id:     id,
		Tokens: tokens,
	}
}
