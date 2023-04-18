package main

import (
	"os"

	sf "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
)

type FileContext struct {
	sf      sf.SourceFile
	changes []tui.SliceIndex
}

func NewFileContext(sf sf.SourceFile) FileContext {
	return FileContext{sf: sf}
}

func (self *FileContext) Source() *sf.SourceFile {
	return &self.sf
}

func (self *FileContext) Change(index tui.SliceIndex, text string) {
	self.sf.Text().SetSliceText(index, text)
	self.changes = append(self.changes, index)
}

func (self *FileContext) IsChanged() bool {
	return len(self.changes) != 0
}

func (self *FileContext) AddToBackup(b *Backup) {
	b.SetFile(self.sf.Name())
	tb := self.sf.Text()
	originals := make(map[tui.SliceIndex]string)
	for _, w := range self.sf.Words() {
		originals[w.Index] = w.Original
	}
	for _, c := range self.changes {
		b.AddLine(c.Line(), tb, originals)
	}
}

func (self *FileContext) Write() error {
	data := self.sf.String()
	return os.WriteFile(self.sf.Name(), []byte(data), 0o644)
}
