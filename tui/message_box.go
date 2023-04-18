package tui

import (
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"github.com/JaMo42/spellcheck_comments/util"
)

type mbButton struct {
	label     string
	highlight int
	x         int
	end       int
}

type messageBox struct {
	scr          tcell.Screen
	text         string
	buttons      []mbButton
	selected     int
	showSelected bool
	keys         map[rune]int
	rect         Rectangle
	buttonsY     int
	buttonsBegin int
	buttonsEnd   int
}

func MessageBox(scr tcell.Screen, text string, buttons []string, initialSelection int) string {
	keys := map[rune]int{}
	btns := make([]mbButton, len(buttons))
	var highlight int
	for i, label := range buttons {
		for runeIdx, c := range label {
			c = unicode.ToLower(c)
			if c == 'h' || c == 'j' || c == 'k' || c == 'l' {
				continue
			}
			if _, used := keys[c]; !used {
				keys[c] = runeIdx
				highlight = runeIdx
				break
			}
		}
		btns[i] = mbButton{label, highlight, 0, 0}
	}
	mb := messageBox{
		scr:          scr,
		text:         text,
		buttons:      btns,
		selected:     initialSelection,
		showSelected: false,
		keys:         keys,
	}
	return mb.Run()
}

func (self *messageBox) Run() string {
	self.Layout()
	self.Redraw()
	self.scr.Show()
	maxSelected := len(self.buttons) - 1
	changeSelected := func(by int) {
		sel := util.Clamp(self.selected+by, 0, maxSelected)
		if sel != self.selected || !self.showSelected {
			self.showSelected = true
			self.Select(sel)
			self.scr.Show()
		}
	}
	for {
		ev := self.scr.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			k, r := TranslateControls(ev)
			switch k {
			case tcell.KeyEnter:
				return self.buttons[self.selected].label
			case tcell.KeyLeft:
				changeSelected(-1)
			case tcell.KeyRight:
				changeSelected(1)
			case tcell.KeyEscape:
				return ""
			case tcell.KeyRune:
				button, valid := self.keys[r]
				if valid {
					return self.buttons[button].label
				}
			}
		case *tcell.EventMouse:
			x, y := ev.Position()
			if y == self.buttonsY && x >= self.buttonsBegin && x < self.buttonsEnd {
				for i, b := range self.buttons {
					if x >= b.x && x < b.end {
						if i != self.selected {
							self.showSelected = true
							self.Select(i)
							self.scr.Show()
						}
						if ev.Buttons()&tcell.Button1 == tcell.Button1 {
							return b.label
						}
					}
				}
			}
		case *tcell.EventResize:
			self.Layout()
			self.Redraw()
			self.scr.Show()
		}
	}
}

func (self *messageBox) Layout() {
	screenWidth, screenHeight := self.scr.Size()
	contentWidth := 2 + util.Sum(util.Map(self.buttons, func(b mbButton) int {
		return 2 + runewidth.StringWidth(b.label)
	}))
	if textWidth := runewidth.StringWidth(self.text); textWidth > contentWidth {
		contentWidth = textWidth
	}
	width := contentWidth + 2
	height := 5
	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2
	self.rect = NewRectangle(x, y, width, height)
	x = self.rect.Right() - 1
	self.buttonsY = y + height - 2
	self.buttonsEnd = x
	for i := len(self.buttons) - 1; i >= 0; i-- {
		b := &self.buttons[i]
		b.end = x
		x -= 2 + runewidth.StringWidth(b.label)
		b.x = x
	}
	self.buttonsBegin = x
}

func (self *messageBox) Redraw() {
	x, y, width, height := self.rect.Parts()
	Box(self.scr, x, y, width, height, Colors.BoxOutline)
	FillRect(self.scr, x+1, y+1, width-2, height-2, ' ', tcell.StyleDefault)
	Text(self.scr, x+1, y+1, self.text, tcell.StyleDefault)
	for i := range self.buttons {
		self.DrawButton(i)
	}
}

func (self *messageBox) DrawButton(idx int) {
	button := &self.buttons[idx]
	x := button.x
	var normalStyle, highlightStyle tcell.Style
	if idx == self.selected && self.showSelected {
		normalStyle = tcell.StyleDefault.Reverse(true)
		highlightStyle = normalStyle.Background(tcell.ColorRed)
	} else {
		normalStyle = tcell.StyleDefault
		highlightStyle = normalStyle.Foreground(tcell.ColorRed)
	}
	self.scr.SetContent(x, self.buttonsY, ' ', nil, normalStyle)
	x++
	x = TextWithHighlight(
		self.scr,
		x,
		self.buttonsY,
		button.label,
		button.highlight,
		normalStyle,
		highlightStyle,
	)
	self.scr.SetContent(x, self.buttonsY, ' ', nil, normalStyle)
}

func (self *messageBox) Select(idx int) {
	old := self.selected
	self.selected = idx
	self.DrawButton(old)
	self.DrawButton(idx)
}

func AskYesNo(scr tcell.Screen, text string) bool {
	yes := "Yes"
	no := "No"
	return MessageBox(scr, text, []string{yes, no}, 1) == yes
}
