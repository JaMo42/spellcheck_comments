package tui

import (
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/JaMo42/spellcheck_comments/util"
)

type GetLineState struct {
	history   []string
	lastEvent time.Time
}

func (self *GetLineState) WriteRune(char rune) {
	s := self.String() + string(char)
	self.history = append(self.history, s)
}

func (self *GetLineState) Backspace() bool {
	s := self.String()
	if len(s) == 0 || len(self.history) == 0 {
		return false
	}
	s = s[:len(s)-1]
	self.history = append(self.history, s)
	return true
}

func (self *GetLineState) Clear() bool {
	if len(self.String()) == 0 || len(self.history) == 0 {
		return false
	}
	self.history = append(self.history, "")
	return true
}

func (self *GetLineState) Undo() bool {
	if len(self.history) == 0 {
		return false
	}
	self.history = self.history[:len(self.history)-1]
	return true
}

func (self *GetLineState) String() string {
	if len(self.history) == 0 {
		return ""
	}
	return *util.Back(self.history)
}

// Display returns the string to display the state and a character to highlight.
func (self *GetLineState) Display(width int) (string, int) {
	s := self.String()
	if len(s) >= width {
		skip := len(s) - width + 2
		return "<" + s[skip:], 0
	}
	return s, -1
}

// Event processes a key event, returning true if a redraw is necessary.
func (self *GetLineState) Event(event *tcell.EventKey) bool {
	switch event.Key() {
	// Note: Backspace2 is 'KeyDEL' i.e. forward delete which we don't want here
	// but my terminal sends this for backspace...
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		// FIXME: a ctrl+backspace sequence does not seem to exist.
		if event.Modifiers()&tcell.ModCtrl == tcell.ModCtrl {
			return self.Clear()
		} else {
			return self.Backspace()
		}

	case tcell.KeyCtrlZ:
		return self.Undo()

	case tcell.KeyRune:
		self.WriteRune(event.Rune())
		return true
	}
	return false
}

// MergeHistory merges the last addition to the history if it's close in time
// to the last non-merged change.
func (self *GetLineState) MergeHistory(eventTime time.Time) {
	if eventTime.UnixMilli()-self.lastEvent.UnixMilli() < 500 {
		if len(self.history) >= 2 {
			idx := len(self.history) - 2
			self.history = append(self.history[:idx], self.history[idx+1:]...)
		}
	} else {
		self.lastEvent = eventTime
	}
}
