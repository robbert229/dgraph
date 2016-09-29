package tok

import (
	"log"
	"testing"
)

func TestBasic(t *testing.T) {
	testData := []struct {
		in       string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"  hello  \n  world  ", []string{"hello", "world"}},
		{"在新加坡鞭刑是處置犯人  的方法之一", []string{"在", "新加坡", "鞭刑", "是", "處置", "犯人", "的", "方法", "之一"}},
		{"cafés\ncool", []string{"cafes", "cool"}},
	}

	for _, d := range testData {
		func(in string, expected []string) {
			tokenizer, err := NewTokenizer([]byte(d.in))
			defer tokenizer.Destroy()

			if err != nil {
				t.Error(err)
				return
			}

			var tokens []string
			for {
				sPtr := tokenizer.Next()
				if sPtr == nil {
					break
				}
				tokens = append(tokens, *sPtr)
			}

			log.Println(tokens)
			if len(tokens) != len(expected) {
				t.Errorf("Wrong number of tokens: %d vs %d", len(tokens), len(expected))
				return
			}
			for i := 0; i < len(tokens); i++ {
				if tokens[i] != expected[i] {
					t.Errorf("Expected token [%s] but got [%s]", expected[i], tokens[i])
					return
				}
			}
		}(d.in, d.expected)
	}
}
