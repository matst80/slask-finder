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
		res[c] = token
		c++
	}
	return res
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
