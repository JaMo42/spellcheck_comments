package tui

import (
	"github.com/JaMo42/spellcheck_comments/util"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type TextSlice struct {
	text  string
	style tcell.Style
	// offset is the visual offet of the slice from the start of the line
	offset int
	// width is the visual with of the slice
	width int
}

type Line struct {
	slices []TextSlice
	width  int
}

func (self *Line) AddSlice(text string, style tcell.Style) {
	width := runewidth.StringWidth(text)
	self.slices = append(self.slices, TextSlice{text, style, self.width, width})
	self.width += width
}

type SliceIndex struct {
	line, slice int
}

// TextBuffer holds lines of text that are themselves split into slices.
// It also acts as a builder for itself.
type TextBuffer struct {
	lines []Line
	style tcell.Style
}

func NewTextBuffer() TextBuffer {
	return TextBuffer{[]Line{{[]TextSlice{}, 0}}, tcell.StyleDefault}
}

func (self *TextBuffer) SetStyle(style tcell.Style) {
	self.style = style
}

func (self *TextBuffer) AddSlice(slice string) SliceIndex {
	line := util.Back(self.lines)
	sliceIdx := len(line.slices)
	line.AddSlice(slice, self.style)
	return SliceIndex{len(self.lines) - 1, sliceIdx}
}

func (self *TextBuffer) Newline() {
	self.lines = append(self.lines, Line{[]TextSlice{}, 0})
}

func (self *TextBuffer) GetSlice(idx SliceIndex) *TextSlice {
	return &self.lines[idx.line].slices[idx.slice]
}

func (self *TextBuffer) PrintLineAt(scr tcell.Screen, line, x, y int) {
	col := x
	for _, slice := range self.lines[line].slices {
		Text(scr, col, y, slice.text, slice.style)
		col += len(slice.text)
	}
}
