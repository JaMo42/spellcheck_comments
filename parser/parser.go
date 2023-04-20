// Package parser implements routines for reading the input files.
package parser

import (
	"regexp"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/trustmaster/go-aspell"

	. "github.com/JaMo42/spellcheck_comments/common"
	sf "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
)

// Filter returns true if none of the filters match the word.
func Filter(s string, filters []*regexp.Regexp) bool {
	for _, re := range filters {
		if re.MatchString(s) {
			return false
		}
	}
	return true
}

func IsWord(s string) bool {
	letters := 0
	haveApostrophe := false
	for _, char := range s {
		if unicode.IsLetter(char) {
			letters++
		} else if char == '\'' {
			if haveApostrophe {
				return false
			}
			haveApostrophe = true
		}
	}
	return letters >= 2
}

func Parse(
	fileName, source string,
	commentStyle CommentStyle,
	speller aspell.Speller,
	cfg *Config,
	ignoreList *IgnoreList,
	useDefaultCommentColor bool,
) sf.SourceFile {
	_lexer := NewLexer(source, commentStyle)
	lexer := NewPeekable[Token](&_lexer)
	tb := tui.NewTextBuffer(cfg.General.TabSize)
	words := []sf.Word{}
	inComment := false
	dimCode := cfg.General.DimCode
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
			commentColor = tui.Ansi2Style(FallbackCommentColor)
		}
		commentColor = commentColor.Dim(false)
	}
	tb.SetStyle(tcell.StyleDefault.Dim(dimCode))
loop:
	for {
		tok := lexer.Next()
		switch tok.kind {
		case TokenKind.Code:
			tb.AddTabbedSlice(tok.text)

		case TokenKind.CommentWord:
			idx := tb.AddSlice(tok.text)
			if IsWord(tok.text) &&
				!ignoreList.Ignore(tok.text) &&
				!speller.Check(tok.text) &&
				Filter(tok.text, filters) {
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
	// The last line in the text buffer is just an empty list caused by the
	// final newline, we remove it so no additional line number is displayed.
	tb.RemoveLastLineIfEmpty()
	// We need to set the slice pointers after building the text buffer as these
	// point into slices which may be reallocated during creation.
	for i := range words {
		words[i].Slice = tb.GetSlice(words[i].Index)
	}
	return sf.NewSourceFile(fileName, tb, words)
}
