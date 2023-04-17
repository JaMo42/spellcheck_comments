package parser

import (
	"fmt"
	"strings"
	"unicode"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/util"
)

const (
	eofRune            = rune(0)
	eofStateInfo       = -1
	lexStateInCode int = iota
	lexStateInEscape
	lexStateInComment
)

// lexTransition is a state info pair used for switching.
type lexTransition struct{ from, to int }

// TokenKindType is the underlying type for the values in TokenKind.
type TokenKindType int

// TokenKind acts as a namespace for token types.
var TokenKind = struct {
	Code         TokenKindType
	Style        TokenKindType
	CommentBegin TokenKindType
	CommentWord  TokenKindType
	CommentEnd   TokenKindType
	Newline      TokenKindType
	EOF          TokenKindType
}{0, 1, 2, 3, 4, 6, 7}

func LexerTokenKindName(kind TokenKindType) string {
	switch kind {
	case TokenKind.Code:
		return "Code"
	case TokenKind.Style:
		return "Style"
	case TokenKind.CommentBegin:
		return "CommentBegin"
	case TokenKind.CommentWord:
		return "CommentWord"
	case TokenKind.CommentEnd:
		return "CommentEnd"
	case TokenKind.Newline:
		return "Newline"
	case TokenKind.EOF:
		return "EOF"
	}
	panic("not a token kind")
}

type Token struct {
	kind   TokenKindType
	line   int
	column int
	text   string
}

// String returns a string to display the token, use Text() to get the tokens
// text.
func (self *Token) String() string {
	return fmt.Sprintf(
		"%s(%s)",
		LexerTokenKindName(self.kind),
		strings.ReplaceAll(self.text, "\x1b", "\\e"),
	)
}

func (self Token) WithColumn(column int) Token {
	self.column = column
	return self
}

func (self *Token) Kind() TokenKindType {
	return self.kind
}

func (self *Token) Text() string {
	return self.text
}

type Lexer struct {
	source       []rune
	used         int
	line         int
	column       int
	dfa          Dfa
	state        int
	commentState Optional[State]
	ignoreWord   bool
	wordLength   int
	nextTokens   []Token
}

func buildDfa(style CommentStyle) Dfa {
	// TODO: could cache these but it's probably not worth it
	dfa := NewDfa()
	inCodeState := dfa.AddState(lexStateInCode)
	inEscapeState := dfa.AddState(lexStateInEscape)
	inCodeState.AddTransition("\x1b", inEscapeState.Id())
	// We only support SGR sequences so escape sequences always end with `m`.
	inEscapeState.AddTransition("m", inCodeState.Id())
	// All line comment variants can share the same state.
	// Line comments and block comments use the same info as we only need the
	// the name to check if we are in any comment state.
	inLineState := dfa.AddState(lexStateInComment)
	for _, token := range style.Line {
		inCodeState.AddTransition(token, inLineState.Id())
		inLineState.AddTransition("\n", inCodeState.Id())
		inLineState.AddTransition("\x1b", inEscapeState.Id())
	}
	for i, begin := range style.MultiBegin {
		end := style.MultiEnd[i]
		// Each block comments variant needs its own state to ensure we enter
		// and leave the comment with matching tokens (i.e. """ vs '''
		// doc-strings) in Python.
		state := dfa.AddState(lexStateInComment)
		inCodeState.AddTransition(begin, state.Id())
		state.AddTransition(end, inCodeState.Id())
		state.AddTransition("\x1b", inEscapeState.Id())
		state.AddTransition("\n", state.Id())
	}
	inCodeState.AddTransition("\n", inCodeState.Id())
	eofState := dfa.AddState(eofStateInfo)
	inCodeState.AddTransition(string(eofRune), eofState.Id())
	return dfa
}

func NewLexer(source string, commentStyle CommentStyle) Lexer {
	dfa := buildDfa(commentStyle)
	runes := []rune(source)
	runes = append(runes, eofRune)
	return Lexer{
		// TODO: we use runes to get correct column information but since we need
		// to calculate the visual column later anyways we don't actually need
		// line or column information at all and could just use bytes.
		runes,
		0,
		0,
		0,
		dfa,
		lexStateInCode,
		None[State](),
		false,
		0,
		[]Token{},
	}
}

// drop drops count characters from the source.
func (self *Lexer) drop(count int) {
	self.source = self.source[count:]
	self.used -= count
	if self.used < 0 {
		self.used = 0
	}
}

// consume consumes the user text, returning it as a string.
func (self *Lexer) consume(count int) string {
	str := string(self.source[:count])
	self.drop(count)
	return str
}

// createToken creates a token of the given kind with the used text.
// If there is no used text None is returned.
func (self *Lexer) createToken(kind TokenKindType) Optional[Token] {
	if self.used <= 0 {
		return None[Token]()
	}
	text := self.consume(self.used)
	return Some(Token{
		kind,
		self.line,
		// Since newlines are also tokens we don't need to check whether we
		// cross one here.
		self.column - len([]rune(text)),
		text,
	})
}

// createMarker creates an empty token at the current position.
func (self *Lexer) createMarker(kind TokenKindType) Token {
	return Token{
		kind,
		self.line,
		self.column,
		"",
	}
}

func isWordChar(char rune) bool {
	return unicode.IsLetter(char) || char == '-' || char == '\'' || char == '_'
}

// processInComment processes one character inside a comment, adding tokens
// to the internal list.
func (self *Lexer) processInComment(char rune) {
	addToken := func(t Token) {
		self.nextTokens = append(self.nextTokens, t)
	}
	inWord := self.wordLength > 1
	if (char == '@' || char == '\\') && !inWord {
		self.createToken(TokenKind.Code).Then(addToken)
		self.ignoreWord = true
	} else if isWordChar(char) {
		self.wordLength++
	} else if inWord {
		if self.ignoreWord {
			self.ignoreWord = false
		} else {
			self.used -= 1
			self.used -= self.wordLength
			self.createToken(TokenKind.Code).Then(addToken)
			self.used = self.wordLength
			self.createToken(TokenKind.CommentWord).Then(addToken)
			self.used++
		}
		self.wordLength = 0
	} else {
		self.wordLength = 0
	}
}

// getNextTokens processes the source until at least 1 new token is created.
func (self *Lexer) getNextTokens() {
	// Note: regarding the doc comment, we do not stop once we have a token
	// inside a comment but only on a state change of the DFA.
	addToken := func(t Token) {
		t.column -= self.used
		self.nextTokens = append(self.nextTokens, t)
	}
	lastState := self.dfa.CurrentState()
	if len(self.source) == 0 {
		addToken(self.createMarker(TokenKind.EOF))
		return
	}
	for {
		char := self.source[self.used]
		self.used++
		self.column++
		if self.state == lexStateInComment {
			self.processInComment(char)
		}
		stateChanged, tokenLength := self.dfa.Process(char)
		if stateChanged {
			self.state = self.dfa.CurrentState().info
			if self.state == eofStateInfo {
				self.nextTokens = append(self.nextTokens, self.createMarker(TokenKind.EOF))
				return
			}
			transition := lexTransition{lastState.info, self.state}
			switch transition {
			case lexTransition{lexStateInCode, lexStateInComment}:
				self.used -= tokenLength
				self.createToken(TokenKind.Code).Then(addToken)
				addToken(self.createMarker(TokenKind.CommentBegin))
				// TODO: I don't think this is neccessary
				//self.used = tokenLength
				//self.createToken(TokenKind.Code).Then(addToken)

			case lexTransition{lexStateInCode, lexStateInEscape}:
				self.used -= tokenLength
				self.createToken(TokenKind.Code).Then(addToken)

			case lexTransition{lexStateInComment, lexStateInCode}:
				if char == '\n' {
					self.used -= 1
				}
				self.createToken(TokenKind.Code).Then(addToken)
				addToken(self.createMarker(TokenKind.CommentEnd))
				if char == '\n' {
					self.drop(1)
					addToken(self.createMarker(TokenKind.Newline))
				}

			case lexTransition{lexStateInEscape, lexStateInCode}:
				addToken(self.createToken(TokenKind.Style).Unwrap())
				self.commentState.Take().Then(func(id State) {
					// When finishing a escape sequence we always go back to the code so if there
					// was a escape sequence in a comment we need to manually go back to that
					// comment state.
					self.dfa.ForceState(id)
					self.state = self.dfa.CurrentState().info
				})

			// Note: due to the above the `escape -> comment` transition does not exist.

			case lexTransition{lexStateInComment, lexStateInEscape}:
				self.used -= tokenLength
				self.commentState = Some(lastState.id)
				self.createToken(TokenKind.Code).Then(addToken)

			case lexTransition{lexStateInCode, lexStateInCode}:
				fallthrough
			case lexTransition{lexStateInComment, lexStateInComment}:
				// Caused by a newline.
				self.used -= 1
				self.createToken(TokenKind.Code).Then(addToken)
				self.drop(1)
				addToken(self.createMarker(TokenKind.Newline))
				self.line++
				self.column = 0
			}
			break
		}
	}
}

// Next returns the next token. If the input is exhausted all calls return EOF.
func (self *Lexer) Next() (t Token) {
	for len(self.nextTokens) == 0 {
		self.getNextTokens()
	}
	t, self.nextTokens = util.Xxs(self.nextTokens)
	return t
}
