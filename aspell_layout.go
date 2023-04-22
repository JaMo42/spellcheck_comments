package main

import (
	"fmt"
	"path/filepath"

	"github.com/gdamore/tcell/v2"

	. "github.com/JaMo42/spellcheck_comments/common"
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
	highlight       tui.SliceIndex
	bottomStatus    bool
	source          tui.TextBufferView
	dock            tui.Dock
	statusBar       tui.StatusBar
	suggestionCount int
}

func (self *AspellLayout) Configure(cfg *Config) {
	self.bottomStatus = cfg.General.BottomStatus
	self.suggestionCount = cfg.General.Suggestions
}

func (self *AspellLayout) SetSource(sf *SourceFile) {
	self.source.SetTextBuffer(sf.Text())
	self.statusBar.SetLeft(filepath.Clean(sf.Name()))
}

func (self *AspellLayout) Show(index tui.SliceIndex) {
	self.source.ScrollTo(index.Line(), 5, false)
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
	dynRows := util.CeilDiv(self.suggestionCount, columnCount)
	self.dock = tui.NewDock(tui.Alignment.End, tui.Alignment.Fill, columnCount, dynRows, permRows)
	self.dock.SetPermanentItems(globalControls)
	self.dock.TranslateAction(func(group int, item int) any {
		if group == 0 {
			return ActionSelectSuggestion{item}
		} else {
			return globalControls[item].Action()
		}
	})
	self.dock.AlwaysShowSelection(true)
	self.statusBar = tui.NewStausBar()
	self.statusBar.SetRight(fmt.Sprintf("%s %s", appName, appVersion))
}

func (self *AspellLayout) Layout(width, height int) {
	screen := tui.NewRectangle(0, 0, width, height-1)
	if !self.bottomStatus {
		screen.Y++
		self.statusBar.Viewport(0, width)
	} else {
		self.statusBar.Viewport(height-1, width)
	}
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
		self.statusBar.Redraw(scr)
	}
	self.dock.Redraw(scr)
}
