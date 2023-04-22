package parser

import (
	"github.com/JaMo42/spellcheck_comments/util"
)

// State identifier
type State int

// Token that's grown as the DFA processes more bytes.
type dfaToken = uint64

type maskedToken struct {
	Data   dfaToken
	Mask   uint64
	Length int
}

func newMaskedToken(text string) maskedToken {
	length := len([]rune(text))
	mask := (uint64(1) << (len(text) * 8)) - 1
	return maskedToken{
		Data:   util.String2Int(text),
		Mask:   mask,
		Length: length,
	}
}

func (self maskedToken) Eq(token dfaToken) bool {
	// We always check the low bytes of the tokens, this works for us
	// since we reset the token when leaving a comment and don't add
	// anything to it in code so for the starting tokens the DFAs token
	// will only contain the data we want and for ending tokens it works
	// naturally.
	return self.Data == uint64(token)&self.Mask
}

type Transition struct {
	token   maskedToken
	toState State
}

func newTransition(str string, toState State) Transition {
	return Transition{newMaskedToken(str), toState}
}

type DfaState struct {
	id   State
	info int
	// Set of characters this state wants to accept
	useChars    []rune
	transitions []Transition
	// Note: we expect a small number of relevant characters and states
	//       so we use arrays for these instead of maps.
	isRecursive bool
	descent     maskedToken
	ascent      maskedToken
}

func newDfaState(id State, info int) *DfaState {
	state := new(DfaState)
	*state = DfaState{
		id:          id,
		info:        info,
		useChars:    []rune{},
		transitions: []Transition{},
	}
	return state
}

// MakeRecursive turns this state into a recursive state. The descent token
// increases the recursion depth and ascent decreases it. Either of these do
// not count as state changes.
func (self *DfaState) MakeRecursive(descent, ascent string) {
	self.isRecursive = true
	self.descent = newMaskedToken(descent)
	self.ascent = newMaskedToken(ascent)
}

func (self *DfaState) Id() State {
	return self.id
}

func (self *DfaState) AddTransition(token string, toState State) {
	for _, c := range token {
		if !util.Contains(self.useChars, c) {
			self.useChars = append(self.useChars, c)
		}
	}
	self.transitions = append(self.transitions, newTransition(token, toState))
}

type Dfa struct {
	// Need to store as pointer so the value returned from `AddState`
	// stays valid even when this slice is reallocated
	states         []*DfaState
	current        State
	token          dfaToken
	recursionDepth int
}

func NewDfa() Dfa {
	return Dfa{
		states:  []*DfaState{},
		current: 0,
		token:   0,
	}
}

func (self *Dfa) AddState(info int) *DfaState {
	id := len(self.states)
	self.states = append(self.states, newDfaState(State(id), info))
	return self.states[id]
}

func (self *Dfa) CurrentState() *DfaState {
	return self.states[int(self.current)]
}

// Process processes one character of input. Returns whether the current state
// changed and the length in runes of the token that caused the state change.
func (self *Dfa) Process(c rune) (bool, int) {
	currentState := self.CurrentState()
	if !util.Contains(currentState.useChars, c) {
		self.token = 0
		return false, 0
	}
	if c < 0x80 {
		self.token <<= 8
		self.token |= uint64(c)
	} else {
		str := string(c)
		self.token <<= 8 * len(str) // len gives length in bytes
		self.token |= util.String2Int(str)
	}
	if currentState.isRecursive {
		if currentState.descent.Eq(self.token) {
			self.recursionDepth++
			self.token = 0
			return false, 0
		} else if self.recursionDepth != 0 && currentState.ascent.Eq(self.token) {
			self.recursionDepth--
			self.token = 0
			return false, 0
		}
	}
	for _, trans := range currentState.transitions {
		if trans.token.Eq(self.token) {
			self.current = trans.toState
			self.token = 0
			return true, trans.token.Length
		}
	}
	return false, 0
}

// ForceState sets the current state and resets the token.
func (self *Dfa) ForceState(id State) {
	self.current = id
	self.token = 0
}
