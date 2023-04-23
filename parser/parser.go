// Package parser implements routines for reading the input files.
package parser

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/trustmaster/go-aspell"

	. "github.com/JaMo42/spellcheck_comments/common"
	sf "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
)

type CommentRange struct {
	begin tui.SliceIndex
	end   tui.SliceIndex
}

func (self *CommentRange) Contains(index tui.SliceIndex) bool {
	return index.IsSameOrAfter(self.begin) && index.IsBefore(self.end)
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

// IsWord returns true if the string contains at least 2 letters and at most
// one aposthrophe for words like "don't".
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

// TrimSymbols strips the string from leading and trailing ascii punctuation
// characters.
func TrimSymbols(s string) (string, string, string) {
	first := 0
	for first < len(s) && s[first] < 0x80 && unicode.IsPunct(rune(s[first])) {
		first++
	}
	last := len(s) - 1
	if first == last {
		return s, "", ""
	}
	for last >= first && s[last] < 0x80 && unicode.IsPunct(rune(s[last])) {
		last--
	}
	return s[:first], s[first : last+1], s[last+1:]
}

// endCharMap defines scores for the commentIsCode function.
var endCharMap = map[byte]float32{
	';': 1.25,
	'{': 1.25,
	'}': 1.25,
	',': 0.1,
	'(': 1.0,
	')': 0.75,
	'.': -0.5,
	'<': 1.0,
	'>': 1.0,
	'+': 1.0,
	'-': 1.0,
	'*': 1.0,
	'/': 1.0,
	'=': 1.0,
}

func isSpace(s string) bool {
	for _, c := range s {
		if c != ' ' && c != '\t' {
			return false
		}
	}
	return true
}

// commentIsCode checks if the given comment is commented out code.
func commentIsCode(
	begin, end tui.SliceIndex,
	text *tui.TextBuffer,
	lineBeginTokens []string,
) bool {
	target := float32(end.Line()-begin.Line()) * 0.8
	if end.Line() == begin.Line() {
		target = 1.0
	}
	confidence := float32(0.0)
	emptyLine := true
	first := true
	text.ForEachSliceInRange(begin, end, func(slice *tui.TextSlice, eol bool) {
		if text := slice.Text(); !isSpace(text) {
			emptyLine = false
			// Check for a line comment within a block comment, this is very
			// suspicious.
			if !first {
				first = false
				for _, token := range lineBeginTokens {
					if strings.HasPrefix(text, token) {
						confidence += 5.0
						return
					}
				}
			}
		}
		first = false
		if eol {
			if emptyLine {
				confidence += 1.0
				return
			}
			text := slice.Text()
			confidence += endCharMap[text[len(text)-1]]
			emptyLine = true
		}
	})
	return confidence >= target
}

// FilterCommentedCode attemps to identify commented out code and to remove all
// matches words inside those comments. This is based on line ending characters
// and is mainly intended for languages using curly braces and/or semicolons to
// terminate lines.
// This also does not work well in continuous lines comments are used instead
// of block comments as they are all considered separate comments.
func FilterCommentedCode(
	words []sf.Word,
	text *tui.TextBuffer,
	comments []CommentRange,
	lineBeginTokens []string,
) []sf.Word {
	comments = util.Filter(comments, func(comment CommentRange) bool {
		return commentIsCode(comment.begin, comment.end, text, lineBeginTokens)
	})
	if len(comments) == 0 {
		return words
	}
	words = util.StableFilter(words, func(w sf.Word) bool {
		if w.Index.IsSameOrAfter(comments[0].end) {
			comments = comments[1:]
		}
		for _, comment := range comments {
			if comment.Contains(w.Index) {
				return false
			}
		}
		return true
	})
	return words
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
	tb.SetStyle(tcell.StyleDefault.Dim(dimCode))
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
	commentRanges := []CommentRange{}
	var commentBegin tui.SliceIndex

loop:
	for {
		tok := lexer.Next()
		switch tok.kind {
		case TokenKind.Code:
			tb.AddTabbedSlice(tok.text)

		case TokenKind.CommentWord:
			before, word, after := TrimSymbols(tok.text)
			if len(before) > 0 {
				tb.AddSlice(before)
			}
			if len(word) > 0 {
				idx := tb.AddSlice(word)
				if IsWord(word) &&
					!ignoreList.Ignore(word) &&
					!speller.Check(word) &&
					Filter(word, filters) {
					words = append(words, sf.NewWord(word, nil, idx))
				}
			}
			if len(after) > 0 {
				tb.AddSlice(after)
			}

		case TokenKind.CommentBegin:
			if useDefaultCommentColor {
				tb.SetStyle(commentColor)
			}
			inComment = true
			commentBegin = tb.NextIndex()

		case TokenKind.CommentEnd:
			if useDefaultCommentColor {
				tb.SetStyle(tcell.StyleDefault.Dim(dimCode))
			}
			inComment = false
			range_ := CommentRange{commentBegin, tb.NextIndex()}
			// no clue why this is needed but it seems to always work.
			range_.begin.OffsetLine(-1)
			range_.end.OffsetLine(-1)
			commentRanges = append(commentRanges, range_)

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
	// This may seem weird but it's what vim does and I think it looks nicer
	// although it means there is no visual difference between a file with or
	// without a final newline but this is not a text editor so who cares.
	tb.RemoveLastLineIfEmpty()
	if cfg.General.FilterCommentedCode {
		words = FilterCommentedCode(words, &tb, commentRanges, commentStyle.Line)
	}
	// We need to set the slice pointers after building the text buffer as these
	// point into slices which may be reallocated during creation.
	for i := range words {
		words[i].Slice = tb.GetSlice(words[i].Index)
	}
	return sf.NewSourceFile(fileName, tb, words)
}
