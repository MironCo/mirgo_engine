package engine

// Event is a Unity-style multi-cast event system.
// Allows multiple listeners to subscribe to a single event.
type Event struct {
	listeners []func()
}

// AddListener adds a callback to be invoked when the event fires
func (e *Event) AddListener(callback func()) {
	if callback == nil {
		return
	}
	e.listeners = append(e.listeners, callback)
}

// RemoveListener removes a specific callback (not commonly used, but available)
// Note: In Go, function comparison is tricky. This won't work reliably.
// Best practice: just don't remove listeners, or use a different pattern.
func (e *Event) RemoveListener(callback func()) {
	// Function pointers can't be compared in Go, so this is a no-op
	// If you need to remove listeners, consider using a listener ID system
}

// RemoveAllListeners clears all listeners
func (e *Event) RemoveAllListeners() {
	e.listeners = nil
}

// Invoke calls all registered listeners
func (e *Event) Invoke() {
	for _, listener := range e.listeners {
		if listener != nil {
			listener()
		}
	}
}

// GetListenerCount returns the number of registered listeners (for debugging)
func (e *Event) GetListenerCount() int {
	return len(e.listeners)
}

// EventWithArg is a generic event with one argument
type EventWithArg[T any] struct {
	listeners []func(T)
}

func (e *EventWithArg[T]) AddListener(callback func(T)) {
	if callback == nil {
		return
	}
	e.listeners = append(e.listeners, callback)
}

func (e *EventWithArg[T]) RemoveAllListeners() {
	e.listeners = nil
}

func (e *EventWithArg[T]) Invoke(arg T) {
	for _, listener := range e.listeners {
		if listener != nil {
			listener(arg)
		}
	}
}

func (e *EventWithArg[T]) GetListenerCount() int {
	return len(e.listeners)
}
