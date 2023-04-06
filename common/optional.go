package common

// Optional is a single-value container that can either have a value or not.
type Optional[T any] struct {
	inner  T
	isSome bool
}

// None creates an Optional with no value.
func None[T any]() Optional[T] {
	return Optional[T]{isSome: false}
}

// Some create an Optional containing the given value.
func Some[T any](value T) Optional[T] {
	return Optional[T]{value, true}
}

// IsSome returns true if the optional contains a value.
func (self *Optional[T]) IsSome() bool {
	return self.isSome
}

// assertSome panics if the optional does not contain a value.
func (self *Optional[T]) assertSome() {
	if !self.isSome {
		panic("tried to unwrap `None` value")
	}
}

// Unwrap returns the contained value or panics if the Optional is empty.
func (self Optional[T]) Unwrap() T {
	self.assertSome()
	return self.inner
}

// Get is like Unwrap but returns a pointer to the contained value.
func (self *Optional[T]) Get() *T {
	self.assertSome()
	return &self.inner
}

// Take takes the value out of the Optional, leaving the original without a value.
func (self *Optional[T]) Take() Optional[T] {
	new := *self
	var empty T
	self.inner = empty
	self.isSome = false
	return new
}

// Then calles the given function with the contained value, if there is one.
func (self Optional[T]) Then(f func(T)) {
	if self.isSome {
		f(self.inner)
	}
}
