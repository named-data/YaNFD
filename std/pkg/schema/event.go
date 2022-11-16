package schema

// Event is a chain of callback functions for an event.
// The execution order is supposed to be the addition order.
// Note: to implement `comparable`, you need to use pointer type, like *func()
type Event[T comparable] struct {
	val []T
}

// Add a callback. Note that callback should be a *func.
func (e *Event[T]) Add(callback T) {
	e.val = append(e.val, callback)
}

// Remove a callback
// Seems not useful at all. Do we remove it?
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

// Val returns the value of the event. Used by nodes only.
func (e *Event[T]) Val() []T {
	return e.val
}
