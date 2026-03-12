package watcher

import (
	"context"
	"sync"
	"time"
)

// Debouncer batches events and triggers the handler after a delay
type Debouncer struct {
	interval time.Duration
	handler  func(Event)
	timer    *time.Timer
	mu       sync.Mutex
	events   map[string]Event // path -> latest event
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewDebouncer creates a new Debouncer
func NewDebouncer(interval time.Duration, handler func(Event)) *Debouncer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Debouncer{
		interval: interval,
		handler:  handler,
		events:   make(map[string]Event),
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
}

// Add adds an event to the debouncer
func (d *Debouncer) Add(event Event) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Store the latest event for this path
	d.events[event.Path] = event

	// Reset or start the timer
	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.interval, d.flush)
}

// flush triggers the handler for all pending events
func (d *Debouncer) flush() {
	d.mu.Lock()
	events := d.events
	d.events = make(map[string]Event)
	d.mu.Unlock()

	for _, event := range events {
		select {
		case <-d.ctx.Done():
			return
		default:
			d.handler(event)
		}
	}
}

// Stop stops the debouncer
func (d *Debouncer) Stop() {
	d.cancel()

	d.mu.Lock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.mu.Unlock()

	// Flush remaining events
	d.flush()

	close(d.done)
}

// Done returns a channel that's closed when the debouncer stops
func (d *Debouncer) Done() <-chan struct{} {
	return d.done
}
