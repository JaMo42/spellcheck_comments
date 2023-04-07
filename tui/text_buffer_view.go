package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// TextBufferView manages drawing of a TextBuffer
type TextBufferView struct {
	tb               *TextBuffer
	viewport         Rectangle
	scroll           int
	lineNumbersWidth int
	highlightLine    int
}

func NewTextBufferView() TextBufferView {
	return TextBufferView{
		nil,
		NewRectangle(0, 0, 0, 0),
		0,
		0,
		0,
	}
}

func (self *TextBufferView) SetTextBuffer(tb *TextBuffer) {
	self.tb = tb
	self.lineNumbersWidth = len(fmt.Sprintf("%d", len(tb.lines)))
}

func (self *TextBufferView) SetViewport(viewport Rectangle) {
	self.viewport = viewport
}

func (self *TextBufferView) Redraw(scr tcell.Screen) {
	row := self.viewport.y
	end := self.scroll + self.viewport.height
	if end >= len(self.tb.lines) {
		end = len(self.tb.lines)
	}
	col := self.viewport.x + self.lineNumbersWidth + 1
	for line := self.scroll; line < end; line += 1 {
		lineNum := fmt.Sprintf("%*d", self.lineNumbersWidth, line+1)
		var style tcell.Style
		if line == self.highlightLine {
			style = Colors.CurrentLineNumber
		} else {
			style = Colors.LineNumber
		}
		Text(scr, self.viewport.x, row, lineNum, style)
		self.tb.PrintLineAt(scr, line, col, row)
		row += 1
	}
}

// UpdateLine repaints a line if it'c currently inside the viewport.
func (self *TextBufferView) UpdateLine(scr tcell.Screen, line int) {
	if line >= self.scroll && line < self.scroll+self.viewport.height {
		row := self.viewport.y + (line - self.scroll)
		col := self.viewport.x + self.lineNumbersWidth + 1
		self.tb.PrintLineAt(scr, line, col, row)
	}
}

func (self *TextBufferView) ScrollTo(line, linesAbove int) {
	if line < linesAbove {
		linesAbove = line
	}
	self.scroll = line - linesAbove
	self.highlightLine = line
}
