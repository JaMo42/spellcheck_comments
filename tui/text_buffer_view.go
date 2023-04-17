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
	return TextBufferView{}
}

func (self *TextBufferView) SetTextBuffer(tb *TextBuffer) {
	self.tb = tb
	self.lineNumbersWidth = len(fmt.Sprintf("%d", len(tb.lines)))
}

func (self *TextBufferView) Text() *TextBuffer {
	return self.tb
}

func (self *TextBufferView) SetViewport(viewport Rectangle) {
	self.viewport = viewport
}

// SlicePosition returns the (x, y) coordinates of the slice at index in the
// current viewport.
func (self *TextBufferView) SlicePosition(index SliceIndex) (int, int) {
	row := self.viewport.Y + index.line - self.scroll
	col := self.viewport.X + self.lineNumbersWidth + 1 + self.tb.GetSlice(index).offset
	return col, row
}

func (self *TextBufferView) Redraw(scr tcell.Screen) {
	row := self.viewport.Y
	end := self.scroll + self.viewport.Height
	after := 0
	lines := len(self.tb.lines)
	if end >= lines {
		end = lines
		after = lines - end
	}
	col := self.viewport.X + self.lineNumbersWidth + 1
	begin := self.scroll
	if self.scroll < 0 {
		for i := self.scroll; i < 0; i++ {
			Text(scr, self.viewport.X, row, "~", Colors.LineNumber)
			row++
		}
		begin = 0
	}
	for line := begin; line < end; line++ {
		lineNum := fmt.Sprintf("%*d", self.lineNumbersWidth, line+1)
		var style tcell.Style
		if line == self.highlightLine {
			style = Colors.CurrentLineNumber
		} else {
			style = Colors.LineNumber
		}
		Text(scr, self.viewport.X, row, lineNum, style)
		self.tb.PrintLineAt(scr, line, col, row)
		row++
	}
	for i := 0; i < after; i++ {
		Text(scr, self.viewport.X, row, "~", Colors.LineNumber)
		row++
	}
}

// UpdateLine repaints a line if its currently inside the viewport.
func (self *TextBufferView) UpdateLine(scr tcell.Screen, line int) {
	if line >= self.scroll && line < self.scroll+self.viewport.Height {
		row := self.viewport.Y + (line - self.scroll)
		col := self.viewport.X + self.lineNumbersWidth + 1
		self.tb.PrintLineAt(scr, line, col, row)
	}
}

// UpdateSlice repaints a single slice if its currently inside the viewport.
func (self *TextBufferView) UpdateSlice(scr tcell.Screen, index SliceIndex) {
	if index.line >= self.scroll && index.line < self.scroll+self.viewport.Height {
		slice := self.tb.GetSlice(index)
		row := self.viewport.Y + (index.line - self.scroll)
		col := self.viewport.X + self.lineNumbersWidth + 1 + slice.offset
		Text(scr, col, row, slice.text, slice.style)
	}
}

func (self *TextBufferView) Scroll() int {
	return self.scroll
}

func (self *TextBufferView) ScrollTo(line, linesAbove int, forceAbove bool) {
	if line < linesAbove && !forceAbove {
		linesAbove = line
	}
	self.scroll = line - linesAbove
	self.highlightLine = line
}
