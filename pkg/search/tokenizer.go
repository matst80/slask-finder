package search

import (
	"github.com/matst80/slask-finder/pkg/types"
	"slices"
	"strings"
	"unicode"
)

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
	if slices.Contains(*t, token) {
		return
	}
	*t = append(*t, token)
}

// var replacements = []CharReplacement{
// 	{From: "ö", To: "o"},
// 	{From: "ä", To: "a"},
// 	{From: "å", To: "a"},
// 	{From: "é", To: "e"},
// 	{From: "è", To: "e"},
// 	{From: "ê", To: "e"},
// 	{From: "ë", To: "e"},
// 	{From: "ï", To: "i"},
// 	{From: "î", To: "i"},
// 	{From: "ö", To: "o"},
// 	{From: "ô", To: "o"},
// 	{From: "ü", To: "u"},
// 	{From: "û", To: "u"},
// 	{From: "ÿ", To: "y"},
// 	{From: "ç", To: "c"},
// 	{From: "ñ", To: "n"},
// 	{From: "ß", To: "s"},
// 	{From: "æ", To: "a"},
// 	{From: "ø", To: "o"},
// 	{From: "ø", To: "o"},
// }

var commonIssues = map[rune]rune{
	'ö': 'o',
	'ä': 'a',
	'å': 'a',
	'é': 'e',
	'è': 'e',
	'ê': 'e',
	'ë': 'e',
	'ï': 'i',
	'î': 'i',
	'ô': 'o',
	'ü': 'u',
	'û': 'u',
	'ÿ': 'y',
	'ç': 'c',
	'ñ': 'n',
	'ß': 's',
	'æ': 'a',
	'ø': 'o',
	'Ø': 'o',
}

// func replaceCommonIssues(text string) string {
// 	for _, replacement := range replacements {
// 		text = strings.ReplaceAll(text, replacement.From, replacement.To)
// 	}
// 	return text
// }

// func normalize(s string) string {
// 	t := transform.Chain(norm.NFC, runes.Remove(runes.In(unicode.Common)), norm.NFC)
// 	result, _, err := transform.String(t, s)
// 	if err != nil {
// 		return s
// 	}

// 	return result
// }

func NormalizeWord(text string) Token {
	ret := make([]rune, 0, len(text))
	var l rune
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			l = unicode.ToLower(r)
			if replacement, ok := commonIssues[l]; ok {
				l = replacement
			}
			ret = append(ret, l)
		}
	}
	if string(ret) == "iphon" {
		return Token("iphone")
	}
	return Token(ret) //Token(replaceCommonIssues(strings.ToLower(text)))
}

// func getUniqueTokens(tokens []Token) []Token {
// 	unique := make(map[Token]bool)
// 	res := make([]Token, 0, len(unique))
// 	for _, token := range tokens {
// 		if len(token) > 0 {
// 			if _, ok := unique[token]; !ok {
// 				res = append(res, token)
// 			}
// 			unique[token] = true
// 		}
// 	}

// 	return res
// }

func SplitWords(text string, onWord func(word string, count int, last bool) bool) {
	count := 0
	lastSplit := 0
	sp := types.CurrentSettings.SplitWords

	for idx, chr := range text {
		if chr == ' ' || chr == '\n' || chr == '\t' || chr == ',' || chr == ':' || chr == '.' || chr == '!' || chr == '?' || chr == ';' || chr == '(' || chr == ')' || chr == '[' || chr == ']' || chr == '{' || chr == '}' || chr == '"' || chr == '\'' || chr == '/' {
			if idx > lastSplit {
				word := text[lastSplit:idx]
				for _, split := range sp {
					if strings.Contains(word, split) {
						if !onWord(split, count, false) {
							return
						}
						count++
					}
				}

				if !onWord(word, count, false) {
					return
				}
				count++

			}
			lastSplit = idx + 1
		}
	}
	if lastSplit < len(text) {
		onWord(text[lastSplit:], count, true)
	}
}

func (t *Tokenizer) Tokenize(text string, onToken func(token Token, original string, count int, last bool) bool) {
	//parts := make([]Token, 0, t.MaxTokens)
	tokenNumber := 0
	found := map[Token]struct{}{}
	SplitWords(text, func(word string, count int, last bool) bool {

		normalized := NormalizeWord(word)
		if len(normalized) == 0 {
			return true
		}
		_, hasWord := found[Token(word)]
		if !hasWord {
			onToken(normalized, word, tokenNumber, last)
			tokenNumber++
		}
		found[normalized] = struct{}{}

		return count < t.MaxTokens
	})

	//return getUniqueTokens(parts)
}

// func (t *Tokenizer) MakeDocument(id uint, text ...string) *Document {

// 	tokens := []Token{}
// 	for _, txt := range text {
// 		tokens = append(tokens, t.Tokenize(txt)...)
// 	}
// 	return &Document{
// 		Id:     id,
// 		Tokens: tokens,
// 	}
// }
