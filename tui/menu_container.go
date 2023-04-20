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

func (self *MenuContainer) doEvade() Optional[int] {
	above := None[int]()
	self.evade.Then(func(rect Rectangle) {
		above = self.menu.Evade(rect, self.viewport)
	})
	return above
}

func (self *MenuContainer) SetEvade(evade Optional[Rectangle]) Optional[int] {
	self.evade = evade
	return self.doEvade()
}

func (self *MenuContainer) SetMenuPosition(x, y, wordWidth int) Optional[int] {
	self.menu.SetPosition(
		self.viewport.X+x,
		self.viewport.Y+y,
		wordWidth,
		self.viewport,
		!self.evade.IsSome(),
	)
	self.doEvade()
	return self.doEvade()
}
