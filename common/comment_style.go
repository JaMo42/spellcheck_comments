package common

import (
	"fmt"
	"strings"
)

type StringStyle struct {
	Begin  string `toml:"begin"`
	End    string `toml:"end"`
	Escape string `toml:"escape"`
}

type CommentStyle struct {
	Line         []string      `toml:"line"`
	BlockBegin   []string      `toml:"block-begin"`
	BlockEnd     []string      `toml:"block-end"`
	BlockNesting bool          `toml:"block-nesting"`
	Strings      []StringStyle `toml:"strings"`
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
	if len(self.BlockBegin) != len(self.BlockEnd) {
		return fmt.Errorf("multi-begin and multi-end values do not match")
	}
	for _, field := range [][]string{self.Line, self.BlockBegin, self.BlockEnd} {
		if err := checkTokenLengths(field); err != nil {
			return err
		}
	}
	return nil
}

func (self *CommentStyle) Dump(name string, extensions []string) {
	fmt.Printf("\x1b[1m%s\x1b[m\n", name)
	last := len(self.Line) - 1
	if last >= 0 {
		fmt.Print("   Line styles: ")
		for i, s := range self.Line {
			fmt.Printf("%s%s\x1b[m", FallbackCommentColor, s)
			if i != last {
				fmt.Print(", ")
			}
		}
		fmt.Println()
	}
	last = len(self.BlockBegin) - 1
	if last >= 0 {
		fmt.Print("  Block styles: ")
		for i, begin := range self.BlockBegin {
			end := self.BlockEnd[i]
			fmt.Printf(
				"%s%s\x1b[;2m...\x1b[22m%s%s\x1b[m",
				FallbackCommentColor,
				begin,
				FallbackCommentColor,
				end,
			)
			if i != last {
				fmt.Print(", ")
			}
		}
		fmt.Println()
	}
	last = len(self.Strings) - 1
	if last >= 0 {
		fmt.Print("       Strings: ")
		for i, s := range self.Strings {
			fmt.Printf(
				"%s%s\x1b[;2m...\x1b[22m%s%s\x1b[m",
				FallbackCommentColor,
				s.Begin,
				FallbackCommentColor,
				s.End,
			)
			if i != last {
				fmt.Print(", ")
			}
		}
		fmt.Println()
	}
	if len(extensions) == 1 {
		fmt.Print("     Extension: ")
	} else {
		fmt.Print("    Extensions: ")
	}
	fmt.Println(strings.Join(extensions, ", "))
}
