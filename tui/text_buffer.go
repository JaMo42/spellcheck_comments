package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"github.com/JaMo42/spellcheck_comments/util"
)

type TextSlice struct {
	text  string
	style tcell.Style
	// offset is the visual offet of the slice from the start of the line
	offset int
	// width is the visual with of the slice
	width int
}

func (self *TextSlice) Width() int {
	return self.width
}

// ReverseColors toggles the revcrse attribute of the slices style.
func (self *TextSlice) ReverseColors() {
	_, _, attrs := self.style.Decompose()
	on := attrs&tcell.AttrReverse == tcell.AttrReverse
	self.style = self.style.Reverse(!on)
}

type Line struct {
	slices []TextSlice
	width  int
}

func (self *Line) addSlice(text string, style tcell.Style) {
	width := runewidth.StringWidth(text)
	self.slices = append(self.slices, TextSlice{text, style, self.width, width})
	self.width += width
}

func (self *Line) computeOffsets() {
	offset := 0
	for i := range self.slices {
		self.slices[i].offset = offset
		offset += self.slices[i].width
	}
}

type SliceIndex struct {
	line, slice int
}

func (self *SliceIndex) Line() int {
	return self.line
}

// TextBuffer holds lines of text that are themselves split into slices.
// It also acts as a builder for itself.
type TextBuffer struct {
	lines []Line
	style tcell.Style
	// capacity holds the number of bytes of all slices immediately after
	// their creation. It is not updated if the content of a slice changes.
	capacity int
}

func NewTextBuffer() TextBuffer {
	return TextBuffer{
		lines:    []Line{{[]TextSlice{}, 0}},
		style:    tcell.StyleDefault,
		capacity: 0,
	}
}

func (self *TextBuffer) SetStyle(style tcell.Style) {
	self.style = style
}

func (self *TextBuffer) AddSlice(slice string) SliceIndex {
	line := util.Back(self.lines)
	sliceIdx := len(line.slices)
	line.addSlice(slice, self.style)
	self.capacity += len(slice)
	return SliceIndex{len(self.lines) - 1, sliceIdx}
}

func (self *TextBuffer) Newline() {
	self.lines = append(self.lines, Line{[]TextSlice{}, 0})
}

func (self *TextBuffer) GetSlice(idx SliceIndex) *TextSlice {
	return &self.lines[idx.line].slices[idx.slice]
}

func (self *TextBuffer) SetSliceText(idx SliceIndex, text string) {
	slice := self.GetSlice(idx)
	self.capacity -= len(slice.text)
	self.capacity += len(text)
	slice.text = text
	slice.width = runewidth.StringWidth(text)
	self.lines[idx.line].computeOffsets()
}

func (self *TextBuffer) PrintLineAt(scr tcell.Screen, line, x, y int) {
	col := x
	for _, slice := range self.lines[line].slices {
		Text(scr, col, y, slice.text, slice.style)
		col += slice.width
	}
}

// RequiredCapacity returns the number of bytes needed to store the text inside
// the text buffer.
func (self *TextBuffer) RequiredCapacity() int {
	return self.capacity + len(self.lines)
}

// ForEach calls the given function for each slice and line ending of the text
// buffer. For newlines it is always called with "\n".
func (self *TextBuffer) ForEach(f func(string)) {
	for _, line := range self.lines {
		for _, slice := range line.slices {
			f(slice.text)
		}
		f("\n")
	}
}

// ForEachInLine calls the given function for each slice in the specified line.
func (self *TextBuffer) ForEachInLine(line int, f func(string)) {
	for _, slice := range self.lines[line].slices {
		f(slice.text)
	}
}
