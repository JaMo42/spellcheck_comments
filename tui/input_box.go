package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	. "github.com/JaMo42/spellcheck_comments/common"
)

var (
	ibControls = []struct{ button, what string }{
		{"Ctrl+Z", "Undo"},
		{"Ctrl+C", "Cancel"},
	}
)

type inputBox struct {
	scr                tcell.Screen
	viewport           Rectangle
	state              GetLineState
	caption            string
	placeholder        string
	suggestionProvider func(string) []string
	suggestions        *ListView
	inputFocused       bool
	suggestionCount    int
}

func InputBox(
	scr tcell.Screen, caption, placeholder string, suggest func(string) []string,
) Optional[string] {
	var suggestions *ListView = nil
	if suggest != nil {
		suggestions = new(ListView)
		*suggestions = NewGenericListView(1)
		suggestions.TranslateAction(func(_, item int) any {
			items := suggestions.selectedColumn().items
			if len(items) == 0 {
				return None[string]()
			} else {
				return Some(items[item].label)
			}
		})
		suggestions.AddGroup(5)
	}
	ib := inputBox{
		scr:                scr,
		caption:            caption,
		placeholder:        placeholder,
		suggestionProvider: suggest,
		suggestions:        suggestions,
		inputFocused:       true,
	}
	return ib.Run()
}

func (self *inputBox) Run() Optional[string] {
	self.Layout()
	self.Redraw()
	self.scr.Show()
	for {
		ev := self.scr.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			var finished bool
			var s Optional[string]
			if self.inputFocused {
				finished, s = self.keyInInput(ev)
			} else {
				finished, s = self.keyInSuggestions(ev)
			}
			if finished {
				self.scr.ShowCursor(-1, -1)
				return s
			}

		case *tcell.EventMouse:
			if self.suggestions == nil {
				continue
			}
			x, y := ev.Position()
			if ev.Buttons() == 0 {
				if self.suggestions.Motion(x, y) {
					self.suggestions.Redraw(self.scr, self.viewport.X+1, self.viewport.Y+4)
					self.scr.Show()
				}
			} else if ev.Buttons()&tcell.Button1 == tcell.Button1 {
				if sel := self.suggestions.Click(x, y, ev.Buttons()); sel.IsSome() {
					return sel.Unwrap().(Optional[string])
				}
			}

		case *tcell.EventResize:
			self.Layout()
			self.Redraw()
			self.scr.Show()
		}
	}
}

func (self *inputBox) keyInInput(ev *tcell.EventKey) (bool, Optional[string]) {
	switch ev.Key() {
	case tcell.KeyEnter:
		return true, Some(self.state.String())

	case tcell.KeyCtrlC:
		return true, None[string]()

	case tcell.KeyDown:
		if self.suggestionCount > 0 {
			self.inputFocused = false
			self.suggestions.AlwaysShowSelection = true
			self.scr.ShowCursor(-1, -1)
			self.UpdateText()
			self.scr.Show()
		}

	default:
		if self.state.Event(ev) {
			self.UpdateText()
			self.scr.Show()
		}
		self.state.MergeHistory(ev.When())
	}
	return false, None[string]()
}

func (self *inputBox) keyInSuggestions(ev *tcell.EventKey) (bool, Optional[string]) {
	k, _ := TranslateControls(ev)
	switch k {
	case tcell.KeyUp:
		if self.suggestions.selRow == 0 {
			self.inputFocused = true
			self.suggestions.AlwaysShowSelection = false
			self.UpdateText()
			self.scr.Show()
			return false, None[string]()
		} else {
			self.suggestions.Up()
		}

	case tcell.KeyDown:
		self.suggestions.Down()

	case tcell.KeyEnter:
		return true, self.suggestions.GetSelected().(Optional[string])

	case tcell.KeyCtrlC:
		return true, None[string]()

	default:
		return false, None[string]()
	}
	self.suggestions.Redraw(self.scr, self.viewport.X+1, self.viewport.Y+4)
	self.scr.Show()
	return false, None[string]()
}

func (self *inputBox) Layout() {
	screenWidth, screenHeight := self.scr.Size()
	contentHeight := 3 + 1 + len(ibControls)
	if self.suggestionProvider != nil {
		contentHeight += 5
	}
	width := screenWidth / 3
	height := contentHeight + 2
	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2
	self.viewport = NewRectangle(x, y, width, height)
	if self.suggestions != nil {
		self.suggestions.SetColumnWidth(Some(width - 2))
	}
}

func (self *inputBox) Redraw() {
	self.DrawBox()
	self.UpdateText()
	x := self.viewport.X + 1
	y := self.viewport.Bottom() - 2 - len(ibControls)
	width := self.viewport.Width - 2
	self.scr.SetContent(x-1, y, boxStyle.VerticalRight, nil, Colors.BoxOutline)
	HLine(self.scr, x, y, width, boxStyle.Horizontal, Colors.BoxOutline)
	self.scr.SetContent(x+width, y, boxStyle.VerticalLeft, nil, Colors.BoxOutline)
	for _, control := range ibControls {
		y++
		Text(self.scr, x, y, control.button, tcell.StyleDefault)
		RightText(self.scr, x, y, width, control.what, tcell.StyleDefault)
	}
}

func (self *inputBox) DrawBox() {
	x, y, width, height := self.viewport.Parts()
	captionWidth := runewidth.StringWidth(self.caption)
	Box(self.scr, x, y, width, height, Colors.BoxOutline)
	FillRect(self.scr, x+1, y+1, width-2, height-2, ' ', tcell.StyleDefault)
	self.scr.SetContent(x+2, y, boxStyle.VerticalLeft, nil, Colors.BoxOutline)
	self.scr.SetContent(x+2+captionWidth+1, y, boxStyle.VerticalRight, nil, Colors.BoxOutline)
	Text(self.scr, x+3, y, self.caption, tcell.StyleDefault)
}

func (self *inputBox) UpdateText() {
	x := self.viewport.X + 2
	y := self.viewport.Y + 2
	width := self.viewport.Width - 4
	var outlineStyle tcell.Style
	if self.inputFocused {
		outlineStyle = tcell.StyleDefault.Foreground(tcell.PaletteColor(251))
	} else {
		outlineStyle = tcell.StyleDefault.Foreground(tcell.PaletteColor(243))
	}
	Box(self.scr, x-1, y-1, width+2, 3, outlineStyle)
	text, highlight := self.state.Display(width)
	if len(text) != 0 {
		HLine(self.scr, x, y, width, ' ', tcell.StyleDefault)
		highlightStyle := tcell.StyleDefault.Foreground(tcell.ColorBlue)
		cx := TextWithHighlight(
			self.scr, x, y, text, highlight, tcell.StyleDefault, highlightStyle,
		)
		self.scr.ShowCursor(cx, y)
	} else {
		Text(self.scr, x, y, self.placeholder, tcell.StyleDefault.Dim(true))
		self.scr.ShowCursor(x, y)
	}
	x = self.viewport.X + 1
	y = self.viewport.Y + 4
	width = self.viewport.Width - 2
	if self.suggestions != nil {
		s := self.state.String()
		if len(s) == 0 {
			FillRect(self.scr, x, y, width, 5, ' ', tcell.StyleDefault)
			return
		}
		suggestions := self.suggestionProvider(s)
		self.suggestions.ClearGroup(0)
		if len(suggestions) == 0 {
			FillRect(self.scr, x, y, width, 5, ' ', tcell.StyleDefault)
			return
		}
		// No need to show the input as suggestion.
		if suggestions[0] == s {
			suggestions = suggestions[1:]
		}
		if len(suggestions) > 5 {
			suggestions = suggestions[:5]
		}
		for i, suggestion := range suggestions {
			self.suggestions.AddItem(0, i, '\000', suggestion)
		}
		self.suggestions.SetPosition(x, y)
		self.suggestions.Redraw(self.scr, x, y)
		self.suggestionCount = len(suggestions)
	}
}
