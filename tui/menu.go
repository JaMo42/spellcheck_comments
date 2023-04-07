package tui

import (
	"fmt"

	"github.com/JaMo42/spellcheck_comments/util"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type menuColumn struct {
	items       []int
	itemsWidth  int
	numberWidth int
	isLast      bool
}

func (self *menuColumn) Push(idx int, width int) {
	self.items = append(self.items, idx)
	self.itemsWidth = util.Max(self.itemsWidth, width)
}

func (self *menuColumn) Width() int {
	return self.numberWidth + 1 + self.itemsWidth
}

type Menu struct {
	items      []string
	columns    []menuColumn
	emptySlots int // number of empty slots in the last column
	viewport   Rectangle
	selected   int
	// selRow is the row of the selected item
	selRow int
	// prefSelfRow is the row of the selected item after the last up/down movement
	prefSelRow int
	maxRows    int
	maxColumns int
}

func NewMenu() Menu {
	return Menu{}
}

func (self *Menu) SetConstraints(maxRows, maxColumns int) {
	self.maxRows = maxRows
	self.maxColumns = maxColumns
	self.computeColumns()
}

// SetViewport sets the viewport taht menu may exist within. The actual
// rectangle of the viewport is calculated from what it need to display.
func (self *Menu) SetViewport(viewport Rectangle) {
	// FIXME: clamp could reduce the width or height
	self.viewport.Clamp(viewport)
}

func (self *Menu) SetItems(allItems []string) {
	self.items = allItems
	self.computeColumns()
}

func (self *Menu) computeColumns() {
	displayItems := util.Min(len(self.items), self.maxColumns*self.maxRows)
	var columnCount int
	if displayItems <= self.maxRows {
		columnCount = 1
	} else {
		columnCount = util.CeilDiv(displayItems, self.maxRows)
	}
	rowCount := util.CeilDiv(displayItems, columnCount)
	self.columns = make([]menuColumn, columnCount)
	idx := 0
outer:
	for col := 0; col < columnCount; col++ {
		for row := 0; row < rowCount; row++ {
			self.columns[col].Push(idx, runewidth.StringWidth(self.items[idx]))
			idx++
			if idx == len(self.items) {
				break outer
			}
		}
		self.columns[col].numberWidth = len(fmt.Sprintf("%d", idx))
	}
	columnsWidth := util.Sum(util.Map(self.columns, func(c menuColumn) int {
		return c.Width()
	}))
	self.viewport.width = columnsWidth + 2 + 2*(columnCount-1)
	self.viewport.height = rowCount
	self.emptySlots = (rowCount * columnCount) - displayItems
	util.Back(self.columns).isLast = true
}

func (self *Menu) Redraw(scr tcell.Screen) {
	x := self.viewport.x
	var y int
	for _, col := range self.columns {
		for row, item := range col.items {
			style := Colors.Menu.Reverse(item == self.selected)
			y = self.viewport.y + row
			text := fmt.Sprintf(" %*d %-*s ", col.numberWidth, item+1, col.itemsWidth, self.items[item])
			Text(scr, x, y, text, style)
		}
		if col.isLast && self.emptySlots > 0 {
			// XXX: don't know why the extra +1 is needed
			width := col.Width() + 2 + 1
			FillRect(scr, x, y+1, width, self.emptySlots, ' ', Colors.Menu)
		}
		x += col.Width() + 2
	}
}
