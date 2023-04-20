package main

import (
	"github.com/gdamore/tcell/v2"

	. "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
)

// AsRows rearranges the given array so items appear in rows when added to the dock.
func AsRows[T any](items []T, columns int) []T {
	rows := util.CeilDiv(len(items), columns)
	result := make([]T, len(items))
	i := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < columns; c++ {
			idx := c*rows + r
			if idx >= len(items) {
				return result
			}
			result[idx] = items[i]
			i++
		}
	}
	return result
}

type AspellLayout struct {
	highlight tui.SliceIndex
	source    tui.TextBufferView
	dock      tui.Dock
}

func (self *AspellLayout) SetSource(sf *SourceFile) {
	text := sf.Text()
	self.source.SetTextBuffer(text)
	self.highlight = sf.PeekWord().Unwrap().Index
	//text.GetSlice(self.highlight).ReverseColors()
}

func (self *AspellLayout) Show(index tui.SliceIndex) {
	self.source.ScrollTo(index.Line(), 5, false)
	//text := self.source.Text()
	//text.GetSlice(self.highlight).ReverseColors()
	//text.GetSlice(index).ReverseColors()
	self.highlight = index
}

func (self *AspellLayout) SetSuggestions(suggestions []string) {
	self.dock.SetItems(suggestions)
	self.dock.UpdateMouseMap()
}

func (self *AspellLayout) ArrowReceiver() tui.ArrowReceiver {
	return &self.dock
}

func (self *AspellLayout) MouseReceivers() []tui.MouseReceiver {
	return []tui.MouseReceiver{&self.dock}
}

func (self *AspellLayout) Create() {
	const columnCount = 2
	_globalControls := globalControls()
	globalControls := AsRows(_globalControls, columnCount)
	self.source = tui.NewTextBufferView()
	permRows := util.CeilDiv(len(globalControls), columnCount)
	self.dock = tui.NewDock(tui.Alignment.End, tui.Alignment.Fill, columnCount, 5, permRows)
	self.dock.SetPermanentItems(globalControls)
	self.dock.TranslateAction(func(group int, item int) any {
		if group == 0 {
			return ActionSelectSuggestion{item}
		} else {
			return globalControls[item].Action()
		}
	})
	self.dock.AlwaysShowSelection(true)
}

func (self *AspellLayout) Layout(width, height int) {
	screen := tui.NewRectangle(0, 0, width, height)
	self.dock.SetViewport(screen)
	screen.Height = self.dock.Rect().Y
	self.source.SetViewport(screen)
}

func (self *AspellLayout) Update(scr tcell.Screen, widget any) {
	if widget == nil {
		text := self.source.Text()
		text.GetSlice(self.highlight).ReverseColors()
		self.source.Redraw(scr)
		text.GetSlice(self.highlight).ReverseColors()
	}
	self.dock.Redraw(scr)
}
