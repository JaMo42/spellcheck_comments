package parser

import (
	"strings"
	"testing"

	. "github.com/JaMo42/spellcheck_comments/common"
)

var cCommentStyle = CommentStyle{
	Line:       []string{"//"},
	MultiBegin: []string{"/*"},
	MultiEnd:   []string{"*/"},
}

// newToken creates a new token without location info.
func newToken(kind TokenKindType, text ...string) Token {
	return Token{kind, 0, 0, strings.Join(text[:], "")}
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

func TestNewlines(t *testing.T) {
	source := "Line 1\n// Line 2\nLine 3\n/* Line 4\nLine 5 */\nLine 6"
	expected := []Token{
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
		newToken(TokenKind.Newline), // this is not in the source but for
		// now the lexer needs to add it to prevent a crash, this is of course
		// just temporary :^)
		newToken(TokenKind.EOF),
	}
	lexer := NewLexer(source, cCommentStyle)
	ExpectOutput(lexer, expected, tokenInfoEq, t)
}
