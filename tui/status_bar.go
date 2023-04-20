package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type StatusBar struct {
	left  string
	right string
	width int
	y     int
}

func NewStausBar() StatusBar {
	return StatusBar{}
}

func (self *StatusBar) SetLeft(text string) {
	self.left = text
}

func (self *StatusBar) SetRight(text string) {
	self.right = text
}

func (self *StatusBar) Viewport(y, width int) {
	self.y = y
	self.width = width
}

func (self *StatusBar) Redraw(scr tcell.Screen) {
	HLine(scr, 0, self.y, self.width, ' ', Colors.StatusBar)
	Text(scr, 1, self.y, self.left, Colors.StatusBar)
	width := runewidth.StringWidth(self.right)
	Text(scr, self.width-1-width, self.y, self.right, Colors.StatusBar)
}
