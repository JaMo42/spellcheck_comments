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

type DefaultLayout struct {
	highlight       tui.SliceIndex
	bottomStatus    bool
	source          tui.TextBufferView
	pmenu           tui.Menu
	menuContainer   tui.MenuContainer
	globalKeys      tui.Dock
	statusBar       tui.StatusBar
	suggestionCount int
}

func (self *DefaultLayout) Configure(cfg *Config) {
	self.bottomStatus = cfg.General.BottomStatus
	self.suggestionCount = cfg.General.Suggestions
}

func (self *DefaultLayout) SetSource(sf *SourceFile) {
	self.source.SetTextBuffer(sf.Text())
	self.statusBar.SetLeft(filepath.Clean(sf.Name()))
}

func (self *DefaultLayout) Show(index tui.SliceIndex) {
	self.source.ScrollTo(index.Line(), 5, false)
	tb := self.source.Text()
	slice := tb.GetSlice(index)
	x, y := self.source.SlicePosition(index)
	if !self.bottomStatus {
		y--
	}
	self.menuContainer.SetMenuPosition(x, y, slice.Width()).Then(func(above int) {
		self.source.ScrollTo(index.Line(), above, true)
	})
	self.highlight = index
}

func (self *DefaultLayout) SetSuggestions(suggestions []string) {
	self.pmenu.SetItems(suggestions)
}

func (self *DefaultLayout) ArrowReceiver() tui.ArrowReceiver {
	return &self.pmenu
}

func (self *DefaultLayout) MouseReceivers() []tui.MouseReceiver {
	// Note: the order is intentional here, if they end up overlapping the last
	// element in this list will take "precedence" for motion drawing and click
	// order. Note that this is not a layering system but just how the tui is
	// implemented.
	return []tui.MouseReceiver{&self.globalKeys, &self.pmenu}
}

func (self *DefaultLayout) Create() {
	const columnCount = 2
	globalControls := globalControls()
	self.source = tui.NewTextBufferView()
	topGap := 0
	if !self.bottomStatus {
		topGap = 1
	}
	rows := util.CeilDiv(self.suggestionCount, columnCount)
	self.pmenu = tui.NewMenu(tui.MenuLocation.Below, rows, columnCount, topGap)
	self.menuContainer = tui.NewMenuContainer()
	self.menuContainer.SetMenu(&self.pmenu)
	self.globalKeys = tui.NewDock(tui.Alignment.End, tui.Alignment.End, 1, 0, len(globalControls))
	self.globalKeys.SetPermanentItems(globalControls)
	self.pmenu.TranslateAction(func(_, item int) any {
		return ActionSelectSuggestion{item}
	})
	self.globalKeys.TranslateAction(func(_, item int) any {
		return globalControls[item].Action()
	})
	self.statusBar = tui.NewStausBar()
	self.statusBar.SetRight(fmt.Sprintf("%s %s", appName, appVersion))
}

func (self *DefaultLayout) Layout(width, height int) {
	screen := tui.NewRectangle(0, 0, width, height-1)
	if !self.bottomStatus {
		screen.Y++
		self.statusBar.Viewport(0, width)
	} else {
		self.statusBar.Viewport(height-1, width)
	}
	self.source.SetViewport(screen)
	self.menuContainer.SetViewport(screen)
	self.globalKeys.SetViewport(screen)
	self.menuContainer.SetEvade(Some(self.globalKeys.Rect())).Then(func(above int) {
		self.source.ScrollTo(self.highlight.Line(), above, true)
	})
}

func (self *DefaultLayout) Update(scr tcell.Screen, widget any) {
	if widget == nil {
		// We only highlight the current slice on demand so we don't need to
		// worry about any state.
		text := self.source.Text()
		text.GetSlice(self.highlight).ReverseColors()
		self.source.Redraw(scr)
		self.globalKeys.Redraw(scr)
		self.pmenu.Redraw(scr)
		self.source.UpdateSlice(scr, self.highlight)
		text.GetSlice(self.highlight).ReverseColors()
		self.statusBar.Redraw(scr)
	} else if widget == &self.pmenu {
		self.pmenu.Redraw(scr)
	} else if widget == &self.globalKeys {
		self.globalKeys.Redraw(scr)
	}
}
