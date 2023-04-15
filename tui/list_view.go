package tui

import (
	"fmt"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/util"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"golang.org/x/exp/maps"
)

// List view orginization:
//
// lvItem -| 1) abc  3) ghi |- lvGroup |
//           2) def  4) jkl |          |
//           ------                    |
//             |                       |- ListView
//             lvColumn                |
//           a) 123  c) 789            |
//           b) 456                    |
//
// The columns are split into groups to allow "holes" and to easily split
// the columns into a constant and a dynamic part, where only the group
// containing the dynamic items needs to be cleared and repopulated.

type lvItem struct {
	key   rune
	label string
	id    int
}

type lvColumn struct {
	items      []lvItem
	itemsWidth int
	isLast     bool
}

type lvGroup struct {
	columns    []lvColumn
	emptySlots int
	rows       int
}

type lvItemIndex struct {
	group, column, row uint8
}

type lvPoint struct {
	x, y uint16
}

type ListView struct {
	groups              []lvGroup
	columns             int
	itemFormat          string
	itemWidthOffset     int
	columnWidth         Optional[int]
	showSelection       bool
	AlwaysShowSelection bool
	selCol              int
	selRow              int
	selGrp              int
	prefSelRow          int
	style               tcell.Style
	// Due to the split up and irregular nature of this structure we'd need to
	// essentialy check the rectangle of each item to resolve a mouse event but
	// since we don't have that many points in a terminal we can just map each
	// cell occupied by the list view to its index.
	itemPositions   map[lvPoint]lvItemIndex
	translateAction func(int, int) any
}

func NewColumns(
	columnCount int,
	columnWidth Optional[int],
	padColumns bool,
	parenAfterKey bool,
	style tcell.Style,
) ListView {
	var format string
	var itemWidthOffset int
	if parenAfterKey {
		itemWidthOffset = 3
		format = "%c) %-*s"
	} else {
		itemWidthOffset = 2
		format = "%c %-*s"
	}
	if padColumns {
		itemWidthOffset += 2
		format = " " + format + " "
	}
	return ListView{
		columns:         columnCount,
		itemFormat:      format,
		itemWidthOffset: itemWidthOffset,
		columnWidth:     columnWidth,
		style:           style,
		itemPositions:   make(map[lvPoint]lvItemIndex),
	}
}

// TranslateAction specifies how to translate the current selection into an
// action. The parameters are f(groupIndex, itemIndexInGroup).
func (self *ListView) TranslateAction(f func(int, int) any) {
	self.translateAction = f
}

func (self *ListView) AddGroup(rows int) int {
	self.groups = append(self.groups, lvGroup{
		columns:    []lvColumn{{isLast: true}},
		emptySlots: rows,
		rows:       rows,
	})
	return len(self.groups) - 1
}

// SetRows sets the allowed number of rows in a group. This also clears that group.
func (self *ListView) SetRows(group, rows int) {
	self.groups[group].rows = rows
	self.ClearGroup(group)
}

// ClearGroup empties a group.
func (self *ListView) ClearGroup(group int) {
	g := &self.groups[group]
	g.emptySlots = g.rows
	g.columns = []lvColumn{{isLast: true}}
}

func (self *ListView) AddItem(group, id int, key rune, label string) {
	grp := &self.groups[group]
	col := util.Back(grp.columns)
	if len(col.items) == grp.rows {
		if len(grp.columns) == self.columns {
			return
		}
		col.isLast = false
		grp.columns = append(grp.columns, lvColumn{isLast: true})
		grp.emptySlots = grp.rows
		col = util.Back(grp.columns)
	}
	col.items = append(col.items, lvItem{key, label, id})
	grp.emptySlots--
	width := runewidth.StringWidth(label)
	if width > col.itemsWidth {
		col.itemsWidth = width
	}
}

func (self *ListView) Redraw(scr tcell.Screen, x, y int) {
	groupY := y
	baseX := x
	for groupIdx, group := range self.groups {
		for colIdx, col := range group.columns {
			var labelWidth, columnWidth int
			if self.columnWidth.IsSome() {
				columnWidth = self.columnWidth.Unwrap()
				labelWidth = columnWidth - self.itemWidthOffset
			} else {
				labelWidth = col.itemsWidth
				columnWidth = labelWidth + self.itemWidthOffset
			}
			y = groupY
			for rowIdx, item := range col.items {
				style := self.style
				if (self.showSelection || self.AlwaysShowSelection) &&
					groupIdx == self.selGrp &&
					colIdx == self.selCol &&
					rowIdx == self.selRow {
					style = style.Reverse(true)
				}
				width := util.FixPrintfPadding(item.label, labelWidth)
				text := fmt.Sprintf(self.itemFormat, item.key, width, item.label)
				Text(scr, x, y, text, style)
				y++
			}
			if col.isLast && group.emptySlots > 0 {
				FillRect(scr, x, y, columnWidth, group.emptySlots, ' ', self.style)
			}
			x += columnWidth
		}
		groupY += group.rows
		x = baseX
	}
}

// Width returns the combines width of all columns, the fixed column width is
// ignored for this.
func (self *ListView) Width() (width int) {
	for i := 0; i < self.columns; i++ {
		width += util.MaxElem(util.Map(self.groups, func(g lvGroup) int {
			return g.columns[i].itemsWidth + self.itemWidthOffset
		}))
	}
	return width
}

// Height returns the number of rows.
func (self *ListView) Height() (height int) {
	for _, g := range self.groups {
		height += g.rows
	}
	return height
}

func (self *ListView) SetPosition(x, y int) {
	maps.Clear(self.itemPositions)
	yy := uint16(y)
	for gI, g := range self.groups {
		cX := uint16(x)
		for cI, c := range g.columns {
			var width uint16
			if self.columnWidth.IsSome() {
				width = uint16(self.columnWidth.Unwrap())
			} else {
				width = uint16(c.itemsWidth + self.itemWidthOffset)
			}
			for rI := range c.items {
				for iX := uint16(0); iX < width; iX++ {
					point := lvPoint{cX + iX, yy + uint16(rI)}
					index := lvItemIndex{uint8(gI), uint8(cI), uint8(rI)}
					self.itemPositions[point] = index
				}
			}
			cX += width
		}
		yy += uint16(g.rows)
	}
}

func (self *ListView) selectedGroup() *lvGroup {
	return &self.groups[self.selGrp]
}

func (self *ListView) selectedColumn() *lvColumn {
	return &self.selectedGroup().columns[self.selCol]
}

func (self *ListView) Up() {
	if self.selRow == 0 {
		if self.selGrp > 0 {
			self.selGrp--
			g := self.selectedGroup()
			c := &g.columns[self.selCol]
			self.selRow = len(c.items) - 1
		}
	} else {
		self.selRow--
	}
	self.prefSelRow = self.selRow
}

func (self *ListView) Down() {
	if self.selRow+1 < len(self.selectedColumn().items) {
		self.selRow++
		self.prefSelRow = self.selRow
	} else if self.selGrp+1 < len(self.groups) {
		g := &self.groups[self.selGrp+1]
		if self.selCol < len(g.columns) && len(g.columns[self.selCol].items) > 0 {
			self.selGrp++
			self.selRow = 0
		}
	}
	self.prefSelRow = self.selRow
}

func (self *ListView) Left() {
	if self.selCol > 0 {
		self.selCol--
		self.selRow = self.prefSelRow
	}
}

func (self *ListView) Right() {
	if self.selCol+1 < len(self.selectedGroup().columns) {
		self.selCol++
		self.selRow = util.Min(self.selRow, len(self.selectedColumn().items)-1)
	}
}

func (self *ListView) GetSelected() any {
	item := self.selCol*self.selectedGroup().rows + self.selRow
	return self.translateAction(self.selGrp, item)
}

func (self *ListView) Motion(x, y int) bool {
	if index, found := self.itemPositions[lvPoint{uint16(x), uint16(y)}]; found {
		self.selGrp = int(index.group)
		self.selRow = int(index.row)
		self.selCol = int(index.column)
		self.showSelection = true
		return true
	}
	needRedraw := self.showSelection && !self.AlwaysShowSelection
	self.showSelection = false
	return needRedraw
}

func (self *ListView) Click(x, y int, button tcell.ButtonMask) Optional[any] {
	if index, found := self.itemPositions[lvPoint{uint16(x), uint16(y)}]; found {
		item := index.column*uint8(self.groups[index.group].rows) + index.row
		return Some(self.translateAction(int(index.group), int(item)))
	}
	return None[any]()
}
