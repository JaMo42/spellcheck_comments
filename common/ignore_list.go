package common

import (
	"golang.org/x/text/cases"
)

type IgnoreList struct {
	words map[string]bool
	caser *cases.Caser
}

func NewIgnoreList(ignoreCase bool) IgnoreList {
	var caser *cases.Caser
	if ignoreCase {
		caser = new(cases.Caser)
		*caser = cases.Fold()
	}
	return IgnoreList{
		words: make(map[string]bool),
		caser: caser,
	}
}

// transform applies case folding if enabled.
func (self *IgnoreList) transform(word string) string {
	if self.caser != nil {
		word = self.caser.String(word)
	}
	return word
}

// Add adds a word to the list
func (self *IgnoreList) Add(word string) {
	self.words[self.transform(word)] = true
}

// Ignore checks if a word is ignored
func (self *IgnoreList) Ignore(word string) bool {
	return self.words[self.transform(word)]
}
