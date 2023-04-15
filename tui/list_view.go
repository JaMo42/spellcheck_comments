package tui

import (
	"fmt"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/util"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
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

type ListView struct {
	groups        []lvGroup
	columns       int
	parenAfterKey bool
	padColumns    bool
	columnWidth   Optional[int]
	showSelection bool
	selCol        int
	selRow        int
	selGrp        int
	prefSelRow    int
	style         tcell.Style
}

func NewColumns(
	columnCount int,
	columnWidth Optional[int],
	padColumns bool,
	parenAfterKey bool,
	style tcell.Style,
) ListView {
	return ListView{
		columns:       columnCount,
		parenAfterKey: parenAfterKey,
		padColumns:    padColumns,
		columnWidth:   columnWidth,
		style:         style,
	}
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
	widthOffset := 0
	var format string
	if self.parenAfterKey {
		widthOffset = 3
		format = "%c) %-*s"
	} else {
		widthOffset = 2
		format = "%c %-*s"
	}
	if self.padColumns {
		widthOffset += 2
		format = " " + format + " "
	}
	for groupIdx, group := range self.groups {
		for colIdx, col := range group.columns {
			var labelWidth, columnWidth int
			if self.columnWidth.IsSome() {
				columnWidth = self.columnWidth.Unwrap()
				labelWidth = columnWidth - widthOffset
			} else {
				labelWidth = col.itemsWidth
				columnWidth = labelWidth + widthOffset
			}
			y = groupY
			for rowIdx, item := range col.items {
				style := self.style
				if self.showSelection &&
					groupIdx == self.selGrp &&
					colIdx == self.selCol &&
					rowIdx == self.selRow {
					style = style.Reverse(true)
				}
				width := util.FixPrintfPadding(item.label, labelWidth)
				text := fmt.Sprintf(format, item.key, width, item.label)
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
			w := g.columns[i].itemsWidth + 2
			if self.parenAfterKey {
				w += 1
			}
			if self.padColumns {
				w += 2
			}
			return w
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

func (self *ListView) Up() {}

func (self *ListView) Down() {}

func (self *ListView) Left() {}

func (self *ListView) Right() {}

func (self *ListView) GetSelected() any {
	grp := &self.groups[self.selGrp]
	item := self.selCol*grp.rows + self.selRow
	return (uint64(self.selGrp) << 32) | uint64(item)
}

func (self *ListView) Motion(x, y int) bool {
	return false
}

func (self *ListView) Click(button tcell.ButtonMask) Optional[any] {
	return None[any]()
}
