package common

import "fmt"

type CommentStyle struct {
	Line       []string `toml:"line"`
	MultiBegin []string `toml:"multi-begin"`
	MultiEnd   []string `toml:"multi-end"`
}

func checkTokenLengths(tokens []string) error {
	for _, tok := range tokens {
		if len(tok) > 8 {
			return fmt.Errorf("token longer than 8 bytes: %s", tok)
		}
	}
	return nil
}

func (self *CommentStyle) Check() error {
	if len(self.MultiBegin) != len(self.MultiEnd) {
		return fmt.Errorf("multi-begin and multi-end values do not match")
	}
	for _, field := range [][]string{self.Line, self.MultiBegin, self.MultiEnd} {
		if err := checkTokenLengths(field); err != nil {
			return err
		}
	}
	return nil
}
