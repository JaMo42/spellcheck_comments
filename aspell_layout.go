package main

import (
	. "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
	"github.com/JaMo42/spellcheck_comments/util"
	"github.com/gdamore/tcell/v2"
)

// Rearranges the given array so items appear in rows when added to the dock.
func AsRows[T any](items []T, columns int) []T {
	rows := util.CeilDiv(len(items), columns)
	result := make([]T, len(items))
	i := 0
	for c := 0; c < columns; c++ {
		for r := 0; r < rows; r++ {
			idx := r*rows + c
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
	text.GetSlice(self.highlight).ReverseColors()
}

func (self *AspellLayout) Show(index tui.SliceIndex) {
	self.source.ScrollTo(index.Line(), 5, false)
	text := self.source.Text()
	text.GetSlice(self.highlight).ReverseColors()
	text.GetSlice(index).ReverseColors()
	self.highlight = index
}

func (self *AspellLayout) SetSuggestions(suggestions []string) {
	self.dock.SetItems(suggestions)
}

func (self *AspellLayout) ArrowReceiver() tui.ArrowReceiver {
	return &self.dock
}

func (self *AspellLayout) MouseReceivers() []tui.MouseReceiver {
	return []tui.MouseReceiver{&self.dock}
}

func (self *AspellLayout) Create() {
	const columnCount = 2
	globalControls := AsRows(globalControls(), columnCount)
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
}

func (self *AspellLayout) Layout(width, height int) {
	screen := tui.NewRectangle(0, 0, width, height)
	self.dock.SetViewport(screen)
	screen.Height = self.dock.Rect().Y
	self.source.SetViewport(screen)
}

func (self *AspellLayout) Update(scr tcell.Screen, widget any) {
	if widget == nil {
		self.source.Redraw(scr)
	}
	self.dock.Redraw(scr)
}
