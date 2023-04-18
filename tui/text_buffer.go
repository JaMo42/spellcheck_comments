package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"github.com/JaMo42/spellcheck_comments/util"
)

type TextSlice struct {
	text  string
	style tcell.Style
	// offset is the visual offset of the slice from the start of the line
	offset int
	// width is the visual with of the slice
	width int
}

func (self *TextSlice) Width() int {
	return self.width
}

// ReverseColors toggles the reverse attribute of the slices style.
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
	self.addSliceWithWidth(text, width, style)
}

func (self *Line) addSliceWithWidth(text string, width int, style tcell.Style) {
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

func NewSliceIndex(line, slice int) SliceIndex {
	return SliceIndex{line, slice}
}

func (self *SliceIndex) Line() int {
	return self.line
}

// IsAfter returns true if this slice is equal to or after the given slice.
func (self *SliceIndex) IsAfter(other SliceIndex) bool {
	return self.line > other.line || (self.line == other.line && self.slice >= other.slice)
}

// TextBuffer holds lines of text that are themselves split into slices.
// It also acts as a builder for itself.
type TextBuffer struct {
	lines []Line
	style tcell.Style
	// capacity holds the number of bytes of all slices immediately after
	// their creation. It is not updated if the content of a slice changes.
	capacity int
	tabSize  int
}

func NewTextBuffer(tabSize int) TextBuffer {
	return TextBuffer{
		lines:   []Line{{[]TextSlice{}, 0}},
		style:   tcell.StyleDefault,
		tabSize: tabSize,
	}
}

func (self *TextBuffer) SetStyle(style tcell.Style) {
	self.style = style
}

// AddSlice adds a single slice to the buffer. The given slice may not contain
// tab characters.
func (self *TextBuffer) AddSlice(slice string) SliceIndex {
	line := util.Back(self.lines)
	sliceIdx := len(line.slices)
	line.addSlice(slice, self.style)
	self.capacity += len(slice)
	return SliceIndex{len(self.lines) - 1, sliceIdx}
}

// addTabs adds a slice consisting of only tabs to the text buffer.
func (self *TextBuffer) addTabs(count int) {
	line := util.Back(self.lines)
	startingOffset := line.width
	var width int
	// If we're not a multiple of the tab size we need to shorten the shift
	// width of the first tab.
	if startingOffset%self.tabSize == 0 {
		width = count * self.tabSize
	} else {
		width = startingOffset%self.tabSize + (count-1)*self.tabSize
	}
	line.addSliceWithWidth(strings.Repeat("\t", count), width, self.style)
	self.capacity += count
}

// AddTabbedSlice adds a slice that may contain tabs.
func (self *TextBuffer) AddTabbedSlice(text string) {
	var i int
	// repeatedly add alternating slices of tabs and non-tabs
	for len(text) != 0 {
		// tabs
		for i = 0; i < len(text) && text[i] == '\t'; i++ {
		}
		if i > 0 {
			self.addTabs(i)
			text = text[i:]
		}
		// non-tabs
		for i = 0; i < len(text) && text[i] != '\t'; i++ {
		}
		if i > 0 {
			self.AddSlice(text[:i])
			text = text[i:]
		}
	}
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
	first := true
	for _, line := range self.lines {
		if !first {
			f("\n")
		}
		first = false
		for _, slice := range line.slices {
			f(slice.text)
		}
	}
}

// ForEachInLine calls the given function for each slice in the specified line.
func (self *TextBuffer) ForEachInLine(line int, f func(string, SliceIndex)) {
	for i, slice := range self.lines[line].slices {
		f(slice.text, SliceIndex{line, i})
	}
}
