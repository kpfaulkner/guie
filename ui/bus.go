package ui

// EventBus is the optional global event channel. Application code subscribes to
// event types it wants to observe across the whole UI, without wiring a handler
// onto every widget. Every event the framework dispatches is also published
// here. Per-widget callbacks (e.g. Button.OnClick) remain the primary API; the
// bus is for cross-cutting listeners.
//
// Subscribe and publish run on the UI goroutine, so handlers must not block.
type EventBus struct {
	subs map[EventType][]func(Event)
}

func newEventBus() *EventBus {
	return &EventBus{subs: make(map[EventType][]func(Event))}
}

// Subscribe registers fn to be called for every event of type t.
func (b *EventBus) Subscribe(t EventType, fn func(Event)) {
	b.subs[t] = append(b.subs[t], fn)
}

// publish delivers ev to all subscribers of its type.
func (b *EventBus) publish(ev Event) {
	for _, fn := range b.subs[ev.Type] {
		fn(ev)
	}
}
