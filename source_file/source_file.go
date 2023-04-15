// Package source_file contains the SourceFile type. This should ideally be
// in the common package but that would cause an import cycle.
package source_file

import (
	"strings"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/tui"
)

var (
	stringBuilder strings.Builder
)

type Word struct {
	Original string
	Slice    *tui.TextSlice
	Index    tui.SliceIndex
}

func NewWord(original string, slice *tui.TextSlice, index tui.SliceIndex) Word {
	return Word{original, slice, index}
}

type SourceFile struct {
	name     string
	tb       tui.TextBuffer
	words    []Word
	nextWord int
}

func NewSourceFile(name string, tb tui.TextBuffer, words []Word) SourceFile {
	return SourceFile{name, tb, words, 0}
}

func (self *SourceFile) Text() *tui.TextBuffer {
	return &self.tb
}

func (self *SourceFile) Name() string {
	return self.name
}

func (self *SourceFile) Ok() bool {
	return len(self.words) == 0
}

func (self *SourceFile) NextWord() Optional[Word] {
	if self.nextWord == len(self.words) {
		return None[Word]()
	}
	w := self.words[self.nextWord]
	self.nextWord++
	return Some(w)
}

func (self *SourceFile) PeekWord() Optional[Word] {
	if self.nextWord == len(self.words) {
		return None[Word]()
	}
	return Some(self.words[self.nextWord])
}

func (self *SourceFile) String() string {
	stringBuilder.Reset()
	stringBuilder.Grow(self.tb.RequiredCapacity() - stringBuilder.Cap())
	self.tb.ForEach(func(s string) {
		stringBuilder.WriteString(s)
	})
	return stringBuilder.String()
}
