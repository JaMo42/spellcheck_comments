// Package source_file contains the SourceFile type. This should ideally be
// in the common package but that would cause an import cycle.
package source_file

import "github.com/JaMo42/spellcheck_comments/tui"

type Word struct {
	original string
	slice    *tui.TextSlice
	index    tui.SliceIndex
}

type SourceFile struct {
	tb    tui.TextBuffer
	words []Word
}
