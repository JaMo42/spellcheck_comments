package parser

import (
	"strings"
	"testing"

	. "github.com/JaMo42/spellcheck_comments/common"
)

var cCommentStyle = CommentStyle{
	Line:       []string{"//"},
	BlockBegin: []string{"/*"},
	BlockEnd:   []string{"*/"},
	Strings: []StringStyle{
		{Begin: "\"", End: "\"", Escape: "\\\""},
		{Begin: "'", End: "'", Escape: "\\'"},
	},
}

// newToken creates a new token without location info.
func newToken(kind TokenKindType, text ...string) Token {
	return Token{kind, strings.Join(text[:], "")}
}

// tokenInfoEq checks if the kind and text of two tokens are equal
func tokenInfoEq(a, b Token) bool {
	return a.kind == b.kind && a.text == b.text
}

func ExpectOutput(lexer Lexer, expected []Token, eqFn func(a, b Token) bool, t *testing.T) {
	for idx, expectedTok := range expected {
		tok := lexer.Next()
		if !eqFn(tok, expectedTok) {
			t.Errorf(
				"token mismatch at %d: got %s, expected %s",
				idx,
				tok.String(),
				expectedTok.String(),
			)
		}
	}
}

func Expect(t *testing.T, source string, expected []Token, style ...CommentStyle) {
	commentStyle := cCommentStyle
	if len(style) != 0 {
		commentStyle = style[0]
	}
	lexer := NewLexer(source, commentStyle)
	ExpectOutput(lexer, expected, tokenInfoEq, t)
}

func TestOpenComment(t *testing.T) {
	Expect(
		t,
		"/*",
		[]Token{
			newToken(TokenKind.CommentBegin),
			newToken(TokenKind.Code, "/*"),
			newToken(TokenKind.EOF),
		},
	)
}

func TestAlwaysEOFAfterEnd(t *testing.T) {
	Expect(
		t,
		"hello world",
		[]Token{
			newToken(TokenKind.Code, "hello world"),
			newToken(TokenKind.EOF),
			newToken(TokenKind.EOF),
			newToken(TokenKind.EOF),
		},
	)
}

func TestNewlines(t *testing.T) {
	Expect(
		t,
		"Line 1\n// Line 2\nLine 3\n/* Line 4\nLine 5 */\nLine 6",
		[]Token{
			newToken(TokenKind.Code, "Line 1"),
			newToken(TokenKind.Newline),
			newToken(TokenKind.CommentBegin),
			newToken(TokenKind.Code, "// "),
			newToken(TokenKind.CommentWord, "Line"),
			newToken(TokenKind.Code, " 2"),
			newToken(TokenKind.CommentEnd),
			newToken(TokenKind.Newline),
			newToken(TokenKind.Code, "Line 3"),
			newToken(TokenKind.Newline),
			newToken(TokenKind.CommentBegin),
			newToken(TokenKind.Code, "/* "),
			newToken(TokenKind.CommentWord, "Line"),
			newToken(TokenKind.Code, " 4"),
			newToken(TokenKind.Newline),
			newToken(TokenKind.CommentWord, "Line"),
			newToken(TokenKind.Code, " 5 */"),
			newToken(TokenKind.CommentEnd),
			newToken(TokenKind.Newline),
			newToken(TokenKind.Code, "Line 6"),
			newToken(TokenKind.EOF),
		},
	)
}

func TestStrings(t *testing.T) {
	Expect(
		t,
		"//aa\n\"//bb\\\"foo\"//cc",
		[]Token{
			newToken(TokenKind.CommentBegin),
			newToken(TokenKind.Code, "//"),
			newToken(TokenKind.CommentWord, "aa"),
			newToken(TokenKind.CommentEnd),
			newToken(TokenKind.Newline),
			newToken(TokenKind.Code, "\"//bb\\\"foo\""),
			newToken(TokenKind.CommentBegin),
			newToken(TokenKind.Code, "//"),
			newToken(TokenKind.CommentWord, "cc"),
			newToken(TokenKind.EOF),
		},
	)
}

func TestOpenString(t *testing.T) {
	Expect(
		t,
		"'open string",
		[]Token{
			newToken(TokenKind.Code, "'open string"),
			newToken(TokenKind.EOF),
		},
	)
}

func TestNesting(t *testing.T) {
	style := cCommentStyle
	style.BlockNesting = true
	Expect(
		t,
		"/*/*hello*/*/",
		[]Token{
			newToken(TokenKind.CommentBegin),
			newToken(TokenKind.Code, "/*/*"),
			newToken(TokenKind.CommentWord, "hello"),
			newToken(TokenKind.Code, "*/*/"),
			newToken(TokenKind.CommentEnd),
			newToken(TokenKind.EOF),
		},
		style,
	)
}
