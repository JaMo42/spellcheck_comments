// Package parser implements routines for reading the input files.
package parser

import (
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/trustmaster/go-aspell"

	. "github.com/JaMo42/spellcheck_comments/common"
	sf "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
)

func ExpandTabs(s string, tabSize int, b *strings.Builder) string {
	b.Reset()
	for _, c := range []byte(s) {
		if c == '\t' {
			for i := 0; i < tabSize; i++ {
				b.WriteByte(' ')
			}
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// Filter returns true if none of the filters match the word.
func Filter(s string, filters []*regexp.Regexp) bool {
	for _, re := range filters {
		if re.MatchString(s) {
			return false
		}
	}
	return true
}

func Parse(
	fileName, source string,
	commentStyle CommentStyle,
	speller aspell.Speller,
	cfg *Config,
	useDefaultCommentColor bool,
) sf.SourceFile {
	_lexer := NewLexer(source, commentStyle)
	lexer := NewPeekable[Token](&_lexer)
	tb := tui.NewTextBuffer()
	words := []sf.Word{}
	inComment := false
	builder := new(strings.Builder)
	dimCode := cfg.General.DimCode
	tabSize := cfg.General.TabSize
	// Compiling these for every file is fine since we always have the overhead
	// for the first file anyways and the other files are parsed in the background
	// so a minor slowdown doesn't matter.
	filters := util.Map(cfg.General.Filters, func(str string) *regexp.Regexp {
		return regexp.MustCompile(str)
	})
	commentColor := tcell.StyleDefault
	if useDefaultCommentColor {
		if len(cfg.Colors.Comment) != 0 {
			commentColor = tui.Colors.Comment
		} else {
			commentColor = tui.Ansi2Style(FallbackCommentColor).Dim(false)
		}
	}
loop:
	for {
		tok := lexer.Next()
		switch tok.kind {
		case TokenKind.Code:
			tb.AddSlice(ExpandTabs(tok.text, tabSize, builder))

		case TokenKind.CommentWord:
			idx := tb.AddSlice(tok.text)
			if !speller.Check(tok.text) && Filter(tok.text, filters) {
				words = append(words, sf.NewWord(tok.text, nil, idx))
			}

		case TokenKind.CommentBegin:
			if useDefaultCommentColor {
				tb.SetStyle(commentColor)
			}
			inComment = true

		case TokenKind.CommentEnd:
			if useDefaultCommentColor {
				tb.SetStyle(tcell.StyleDefault.Dim(dimCode))
			}
			inComment = false

		case TokenKind.Style:
			style := tui.Ansi2Style(tok.text)
			comment := (inComment || lexer.Peek().kind == TokenKind.CommentBegin) &&
				lexer.Peek().kind != TokenKind.CommentEnd
			if dimCode && !comment {
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
