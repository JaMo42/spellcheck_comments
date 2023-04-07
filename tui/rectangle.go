package tui

type Rectangle struct {
	x, y, width, height int
}

func NewRectangle(x, y, width, height int) Rectangle {
	return Rectangle{x, y, width, height}
}

func (self *Rectangle) Bottom() int {
	return self.y + self.height
}

func (self *Rectangle) Right() int {
	return self.x + self.width
}

func (self *Rectangle) Clamp(inside Rectangle) {
	if self.width > inside.width {
		self.width = inside.width
	}
	if self.height > inside.height {
		self.height = inside.height
	}
	if self.x < inside.x {
		self.x = inside.x
	}
	if self.y < inside.y {
		self.y = inside.y
	}
	if self.Right() > inside.Right() {
		self.x = inside.Right() - self.width
	}
	if self.Bottom() > inside.Bottom() {
		self.y = inside.Bottom() - self.height
	}
}
