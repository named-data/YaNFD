package schema

type Event[T comparable] struct {
	val []T
}

// Add a callback. Note that callback should be a *func.
func (e *Event[T]) Add(callback T) {
	e.val = append(e.val, callback)
}

// Remove a callback
func (e *Event[T]) Remove(callback T) {
	newVal := make([]T, 0, len(e.val))
	for _, v := range e.val {
		if v != callback {
			newVal = append(newVal, v)
		}
	}
	e.val = newVal
}

// NewEvent creates an event. Note that T should be *func.
func NewEvent[T comparable]() *Event[T] {
	return &Event[T]{
		val: make([]T, 0),
	}
}
