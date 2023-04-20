package main

import (
	"os"

	sf "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
)

type FileContext struct {
	sf      sf.SourceFile
	changes map[tui.SliceIndex]bool
}

func NewFileContext(sf sf.SourceFile) FileContext {
	return FileContext{
		sf:      sf,
		changes: make(map[tui.SliceIndex]bool),
	}
}

func (self *FileContext) Source() *sf.SourceFile {
	return &self.sf
}

func (self *FileContext) Word(id int) *sf.Word {
	return &self.sf.Words()[id]
}

// Change changes the text of a slice and adds it to the changes.
func (self *FileContext) Change(index tui.SliceIndex, text string) {
	self.sf.Text().SetSliceText(index, text)
	self.changes[index] = true
}

// RemoveChange removes a slice from the changes and sets its content to the
// given original.
func (self *FileContext) RemoveChange(index tui.SliceIndex, original string) {
	self.sf.Text().SetSliceText(index, original)
	delete(self.changes, index)
}

// SliceIsChanged returns true if the slice with the given index is already changed.
func (self *FileContext) SliceIsChanged(index tui.SliceIndex) bool {
	return self.changes[index]
}

// IsChanged returns true if any changes are made to the file.
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
	for change := range self.changes {
		b.AddLine(change.Line(), tb, originals)
	}
}

func (self *FileContext) Write() error {
	data := self.sf.String()
	return os.WriteFile(self.sf.Name(), []byte(data), 0o644)
}
