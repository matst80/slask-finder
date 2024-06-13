package search

import (
	"testing"
)

func TestTokenizer(t *testing.T) {
	token := Tokenizer{
		MaxTokens: 100,
	}
	res := token.Tokenize("Hello world, how are you?")
	if len(res) != 5 {
		t.Errorf("Expected 5 tokens but got %d", len(res))
	}
	if res[0] != "hello" {
		t.Errorf("Expected 'hello' but got %s", res[0])
	}
	if res[1] != "world" {
		t.Errorf("Expected 'world' but got %s", res[1])
	}
	if res[2] != "how" {
		t.Errorf("Expected 'how' but got %s", res[2])
	}
	if res[3] != "are" {
		t.Errorf("Expected 'are' but got %s", res[3])
	}
	if res[4] != "you" {
		t.Errorf("Expected 'you' but got %s", res[4])
	}
	t.Logf("Result: %v", res)
}

func TestTokenizerDeDuplication(t *testing.T) {
	token := Tokenizer{
		MaxTokens: 100,
	}
	res := token.Tokenize("Hello world, hello world hej hej world")
	if len(res) != 3 {
		t.Errorf("Expected 3 tokens but got %d", len(res))
	}
	if res[0] != "hello" {
		t.Errorf("Expected 'hello' but got %s", res[0])
	}
	if res[1] != "world" {
		t.Errorf("Expected 'world' but got %s", res[1])
	}
	t.Logf("Result: %v", res)
}

func TestCommonCharIssues(t *testing.T) {
	text := "öôüûÿçñßæø"
	res := replaceCommonIssues(text)
	if res != "oouuycnsao" {
		t.Errorf("Expected 'oouuycnsao' but got %s", res)
	}
}

func TestCommonTokenDeDuplication(t *testing.T) {
	token := Tokenizer{
		MaxTokens: 100,
	}
	res := token.Tokenize("öôüûÿçñßæø Öôüûyçñßæø")
	if len(res) != 1 {
		t.Errorf("Expected 1 tokens but got %d", len(res))
	}
	if res[0] != "oouuycnsao" {
		t.Errorf("Expected 'oouuycnsao' but got %s", res[0])
	}
	t.Logf("Result: %v", res)
}
