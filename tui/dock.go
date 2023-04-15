package tui

import (
	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/gdamore/tcell/v2"
)

type DockItem struct {
	text string
	key  Optional[string]
}

type dockColumn struct {
	items      []Optional[int]
	itemsWidth int
}

type Dock struct {
	list     ListView
	vAlign   int
	hAlign   int
	viewport Rectangle
}

func NewDock(vAlign, hAlign, columnCount, dynRows, permRows int) Dock {
	columns := NewColumns(
		columnCount,
		None[int](),
		false,
		true,
		tcell.StyleDefault,
	)
	columns.AddGroup(dynRows)
	columns.AddGroup(permRows)
	return Dock{
		list:   columns,
		vAlign: vAlign,
		hAlign: hAlign,
	}
}

func (self *Dock) SetPermanentItems(items []KeyAction) {
	// XXX: we just KeyAction for the type here as that our only usecase and
	// there is no need to be more generic.
	for i, p := range items {
		self.list.AddItem(1, i, p.Key(), p.Label())
	}
}

func (self *Dock) SetItems(items []string) {
	self.list.ClearGroup(0)
	for i, item := range items {
		key := ' '
		if i < 10 {
			key = rune('0' + (i+1)%10)
		}
		self.list.AddItem(0, i, key, item)
	}
}

func (self *Dock) SetViewport(viewport Rectangle) {
	myWidth := self.list.Width() + 2
	myHeight := self.list.Height() + 2
	xInside, width := alignAxis(viewport.width, myWidth, self.hAlign)
	yInside, height := alignAxis(viewport.height, myHeight, self.vAlign)
	self.viewport = NewRectangle(
		viewport.x+xInside,
		viewport.y+yInside,
		width,
		height,
	)
	self.list.columnWidth = Some((width - 2) / self.list.columns)
}

func (self *Dock) Rect() Rectangle {
	return self.viewport
}

func (self *Dock) Redraw(scr tcell.Screen) {
	x, y, width, height := self.viewport.Parts()
	Box(scr, x, y, width, height, Colors.BoxOutline)
	self.list.Redraw(scr, x+1, y+1)
}
