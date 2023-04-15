package common

// Iterator describes any type with a Next method returning a value. Note that
// Next always returns a value and not an Optional so a specific end value is
// required.
type Iterator[T any] interface {
	Next() T
}

// Peekable wraps an Iterator with the ability to peek the next value.
type Peekable[T any] struct {
	it     Iterator[T]
	peeked Optional[T]
}

// NewPeekable creates a peekable wrapper for the given iterator.
func NewPeekable[T any](it Iterator[T]) Peekable[T] {
	return Peekable[T]{
		it:     it,
		peeked: None[T](),
	}
}

// Peek returns a reference to the next value without advancing the iterator.
func (self *Peekable[T]) Peek() *T {
	if !self.peeked.IsSome() {
		self.peeked = Some(self.it.Next())
	}
	return self.peeked.Get()
}

// Next returns the next value, advancing the iterator.
func (self *Peekable[T]) Next() T {
	if self.peeked.IsSome() {
		return self.peeked.Take().Unwrap()
	}
	return self.it.Next()
}
