// Package parser implements routines for reading the input files.
package parser

import (
	"github.com/trustmaster/go-aspell"

	. "github.com/JaMo42/spellcheck_comments/common"
	sf "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
)

func Parse(
	fileName, source string,
	commentStyle CommentStyle,
	speller aspell.Speller,
	dimCode bool,
) sf.SourceFile {
	_lexer := NewLexer(source, commentStyle)
	lexer := NewPeekable[Token](&_lexer)
	tb := tui.NewTextBuffer()
	words := []sf.Word{}
	inComent := false
loop:
	for {
		tok := lexer.Next()
		switch tok.kind {
		case TokenKind.Code:
			tb.AddSlice(tok.text)

		case TokenKind.CommentWord:
			idx := tb.AddSlice(tok.text)
			if !speller.Check(tok.text) {
				words = append(words, sf.NewWord(tok.text, nil, idx))
			}

		case TokenKind.CommentBegin:
			inComent = true

		case TokenKind.CommentEnd:
			inComent = false

		case TokenKind.Style:
			style := tui.Ansi2Style(tok.text)
			if dimCode && !inComent && lexer.Peek().kind != TokenKind.CommentBegin {
				style = style.Dim(true)
			}
			tb.SetStyle(style)

		case TokenKind.Newline:
			tb.Newline()

		case TokenKind.EOF:
			break loop
		}
	}
	// We need to set the slice pointers after building the text buffer as these
	// point into slices which may be reallocated during creation.
	for i := range words {
		words[i].Slice = tb.GetSlice(words[i].Index)
	}
	return sf.NewSourceFile(fileName, tb, words)
}
