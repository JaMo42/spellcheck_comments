package tui

import . "github.com/JaMo42/spellcheck_comments/common"

// MenuContainer manages the area inside of which a menu may exist.
type MenuContainer struct {
	menu     *Menu
	viewport Rectangle
	evade    Optional[Rectangle]
}

func NewMenuContainer() MenuContainer {
	return MenuContainer{}
}

func (self *MenuContainer) SetMenu(menu *Menu) {
	self.menu = menu
}

func (self *MenuContainer) SetViewport(viewport Rectangle) {
	self.viewport = viewport
}

func (self *MenuContainer) doEvade() {
	self.evade.Then(func(rect Rectangle) {
		self.menu.Evade(rect, self.viewport)
	})
}

func (self *MenuContainer) SetEvade(evade Optional[Rectangle]) {
	self.evade = evade
	self.doEvade()
}

func (self *MenuContainer) SetMenuPosition(x, y, wordWidth int) {
	self.menu.SetPosition(
		self.viewport.x+x,
		self.viewport.y+y,
		wordWidth,
		self.viewport,
		!self.evade.IsSome(),
	)
	self.doEvade()
}
