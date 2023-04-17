package tui

type Rectangle struct {
	X, Y, Width, Height int
}

func NewRectangle(x, y, width, height int) Rectangle {
	return Rectangle{x, y, width, height}
}

func (self *Rectangle) Bottom() int {
	return self.Y + self.Height
}

func (self *Rectangle) Right() int {
	return self.X + self.Width
}

func (self *Rectangle) Clamp(inside Rectangle) {
	if self.Width > inside.Width {
		self.Width = inside.Width
	}
	if self.Height > inside.Height {
		self.Height = inside.Height
	}
	if self.X < inside.X {
		self.X = inside.X
	}
	if self.Y < inside.Y {
		self.Y = inside.Y
	}
	if self.Right() > inside.Right() {
		self.X = inside.Right() - self.Width
	}
	if self.Bottom() > inside.Bottom() {
		self.Y = inside.Bottom() - self.Height
	}
}

func (self *Rectangle) Parts() (int, int, int, int) {
	return self.X, self.Y, self.Width, self.Height
}

func (self *Rectangle) Contains(x, y int) bool {
	return x >= self.X && y >= self.Y && x < self.Right() && y < self.Bottom()
}

func (self *Rectangle) Overlaps(other Rectangle) bool {
	return (self.X <= other.Right() && self.Right() > other.X) &&
		(self.Y <= other.Bottom() && self.Bottom() > other.Y)
}
