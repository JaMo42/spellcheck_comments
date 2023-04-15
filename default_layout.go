package main

import (
	"github.com/gdamore/tcell/v2"

	. "github.com/JaMo42/spellcheck_comments/common"
	. "github.com/JaMo42/spellcheck_comments/source_file"
	"github.com/JaMo42/spellcheck_comments/tui"
)

type DefaultLayout struct {
	highlight     tui.SliceIndex
	source        tui.TextBufferView
	pmenu         tui.Menu
	menuContainer tui.MenuContainer
	globalKeys    tui.Dock
}

func (self *DefaultLayout) SetSource(sf *SourceFile) {
	self.source.SetTextBuffer(sf.Text())
	// Unwrap is safe here as we already skipped the file if it has no wrong words
	self.highlight = sf.PeekWord().Unwrap().Index
	// In the first call to Show ReverseColors will be called twice on the same
	// slice so it needs to already be reversed.
	sf.Text().GetSlice(self.highlight).ReverseColors()
}

func (self *DefaultLayout) Show(index tui.SliceIndex) {
	self.source.ScrollTo(index.Line(), 5, false)
	tb := self.source.Text()
	slice := tb.GetSlice(index)
	x, y := self.source.SlicePosition(index)
	self.menuContainer.SetMenuPosition(x, y, slice.Width())
	tb.GetSlice(self.highlight).ReverseColors()
	slice.ReverseColors()
	self.highlight = index
}

func (self *DefaultLayout) SetSuggestions(suggestions []string) {
	self.pmenu.SetItems(suggestions)
}

func (self *DefaultLayout) ArrowReceiver() tui.ArrowReceiver {
	return &self.pmenu
}

func (self *DefaultLayout) MouseReceivers() []tui.MouseReceiver {
	//return []tui.MouseReceiver{&self.pmenu, &self.globalKeys}
	return []tui.MouseReceiver{}
}

func (self *DefaultLayout) Create() {
	self.source = tui.NewTextBufferView()
	self.pmenu = tui.NewMenu(tui.MenuLocation.Below, 5, 2)
	self.menuContainer = tui.NewMenuContainer()
	self.menuContainer.SetMenu(&self.pmenu)
	self.globalKeys = tui.NewDock(tui.Alignment.End, tui.Alignment.End, 1, 0, 4)
	self.globalKeys.SetPermanentItems(GlobalControls())
}

func (self *DefaultLayout) Layout(width, height int) {
	screen := tui.NewRectangle(0, 0, width, height)
	self.source.SetViewport(screen)
	self.menuContainer.SetViewport(screen)
	self.globalKeys.SetViewport(screen)
	self.menuContainer.SetEvade(Some(self.globalKeys.Rect()))
}

func (self *DefaultLayout) Update(scr tcell.Screen) {
	self.source.Redraw(scr)
	self.globalKeys.Redraw(scr)
	self.pmenu.Redraw(scr)
	self.source.UpdateSlice(scr, self.highlight)
}
