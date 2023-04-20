package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	. "github.com/JaMo42/spellcheck_comments/common"
	"github.com/JaMo42/spellcheck_comments/util"
)

var MenuLocation = struct {
	Right      int
	Below      int
	BelowRight int
}{0, 1, 2}

func MenuLocationFromString(s string) int {
	switch strings.ToLower(s) {
	case "right":
		return MenuLocation.Right
	case "below":
		return MenuLocation.Below
	case "belowright":
		return MenuLocation.BelowRight
	}
	Fatal("invalid menu location: %s", s)
	return -1
}

type Menu struct {
	list              ListView
	viewport          Rectangle
	preferredLocation int
	isBelow           bool
}

func NewMenu(preferredLocation, rows, cols int) Menu {
	list := NewListView(
		cols,
		None[int](),
		true,
		false,
		Colors.Menu,
	)
	list.AlwaysShowSelection = true
	list.AddGroup(rows)
	return Menu{
		list:              list,
		preferredLocation: preferredLocation,
	}
}

func (self *Menu) TranslateAction(f func(int, int) any) {
	self.list.TranslateAction(f)
}

func (self *Menu) SetItems(items []string) {
	self.list.SetRows(0, util.CeilDiv(len(items), 2))
	for i, item := range items {
		key := ' '
		if i < 10 {
			key = rune('0' + (i+1)%10)
		}
		self.list.AddItem(0, i, key, item)
	}
	self.list.ResetSelection()
	self.viewport.Width = self.list.Width()
	self.viewport.Height = self.list.Height()
}

func (self *Menu) Redraw(scr tcell.Screen) {
	self.list.Redraw(scr, self.viewport.X, self.viewport.Y)
}

// SetPosition sets the position on the menu next to a word. x and y are position
// of the word and wordWidth is the width of that word. The menu will be
// appropriately position inside the given rectangle to be next to that word.
// If updatePos is false the list views position is not updated.
func (self *Menu) SetPosition(x, y, wordWidth int, inside Rectangle, updatePos bool) {
	if self.preferredLocation == MenuLocation.Below {
		// Align it so the word and the suggestions are on the same column
		self.viewport.X = x - 3
		self.viewport.Y = y + 1
		self.isBelow = true
	} else if self.preferredLocation == MenuLocation.Right &&
		x+wordWidth+self.viewport.Width <= inside.Right() {
		self.viewport.X = x + wordWidth
		self.viewport.Y = y
		self.isBelow = false
	} else {
		self.viewport.X = x + wordWidth
		self.viewport.Y = y + 1
		self.isBelow = true
	}
	self.viewport.Clamp(inside)
	if updatePos {
		self.list.SetPosition(self.viewport.X, self.viewport.Y)
	}
}

// Evade attempts to evade rect while staying inside inside. The resulting
// viewport will always be inside inside but it cannot guarantee that rect will
// always be evaded. If the menu needs to go above the word, the lines that need
// to be visible above the words line are returned.
// NOTE: rect is assumed to be to the bottom right of the menu.
func (self *Menu) Evade(rect, inside Rectangle) Optional[int] {
	oldX := self.viewport.X
	if self.viewport.Overlaps(rect) {
		if !self.isBelow {
			self.viewport.Y++
			self.isBelow = true
		}
		self.viewport.X = rect.X - self.viewport.Width
		self.viewport.Clamp(inside)
		if self.viewport.Overlaps(rect) {
			// rect was not successfully evaded, need to go above
			self.viewport.Y = 1
			self.viewport.X = oldX
			self.viewport.Clamp(inside)
			self.list.SetPosition(self.viewport.X, self.viewport.Y)
			return Some(self.viewport.Height + 1)
		}
	}
	// We update the list views position even if not changing the our viewport
	// here since if we are calling this function we likely didn't update it
	// in SetPosition.
	self.list.SetPosition(self.viewport.X, self.viewport.Y)
	return None[int]()
}

func (self *Menu) Up() {
	self.list.Up()
}

func (self *Menu) Down() {
	self.list.Down()
}

func (self *Menu) Left() {
	self.list.Left()
}

func (self *Menu) Right() {
	self.list.Right()
}

func (self *Menu) GetSelected() any {
	return self.list.GetSelected()
}

func (self *Menu) Motion(x, y int) bool {
	return self.list.Motion(x, y)
}

func (self *Menu) Click(x, y int, button tcell.ButtonMask) Optional[any] {
	return self.list.Click(x, y, button)
}
