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

func (self *Dock) AlwaysShowSelection(show bool) {
	self.list.AlwaysShowSelection = show
}

func (self *Dock) TranslateAction(f func(int, int) any) {
	self.list.TranslateAction(f)
}

func (self *Dock) SetPermanentItems(items []KeyAction) {
	// XXX: we just KeyAction for the type here as that our only use case and
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
	xInside, width := alignAxis(viewport.Width, myWidth, self.hAlign)
	yInside, height := alignAxis(viewport.Height, myHeight, self.vAlign)
	lastX := self.viewport.X
	lastY := self.viewport.Y
	self.viewport = NewRectangle(
		viewport.X+xInside,
		viewport.Y+yInside,
		width,
		height,
	)
	self.list.columnWidth = Some((width - 2) / self.list.columns)
	if self.viewport.X != lastX || self.viewport.Y != lastY {
		self.list.SetPosition(self.viewport.X+1, self.viewport.Y+1)
	}
}

func (self *Dock) UpdateMouseMap() {
	self.list.SetPosition(self.viewport.X+1, self.viewport.Y+1)
}

func (self *Dock) Rect() Rectangle {
	return self.viewport
}

func (self *Dock) Redraw(scr tcell.Screen) {
	x, y, width, height := self.viewport.Parts()
	Box(scr, x, y, width, height, Colors.BoxOutline)
	self.list.Redraw(scr, x+1, y+1)
}

func (self *Dock) Up() {
	self.list.Up()
}

func (self *Dock) Down() {
	self.list.Down()
}

func (self *Dock) Left() {
	self.list.Left()
}

func (self *Dock) Right() {
	self.list.Right()
}

func (self *Dock) GetSelected() any {
	return self.list.GetSelected()
}

func (self *Dock) Motion(x, y int) bool {
	return self.list.Motion(x, y)
}

func (self *Dock) Click(x, y int, button tcell.ButtonMask) Optional[any] {
	return self.list.Click(x, y, button)
}
