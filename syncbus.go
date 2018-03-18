/*
Package syncbus provides an event bus for testing.

SyncBus can be used to execute test instructions in a concurrent program in a predefined order. It is expected
to be used as a test hook shared between the production and the test code, or left nil if not required.

It provides a wait function for processes in goroutines that should continue only when a predefined signal is
set, or the timeout expires. If the predefined signals are already set at the time of calling wait, the
calling goroutine continues immediately. Once a signal is set, it stays so until it's cleared (reset).

Wait can expect one or more signals represented by keys. The signals don't need to be set simultaneously in
order to release a waiting goroutine. A wait continues once all the signals that it depends on were set.
*/
package syncbus

import (
	"errors"
	"time"
)

type waitItem struct {
	keys     []string
	deadline time.Time
	signal   chan error
}

// SyncBus can be used to synchronize goroutines through signals.
type SyncBus struct {
	timeout  time.Duration
	waiting  []waitItem
	signals  map[string]bool
	wait     chan waitItem
	signal   chan []string
	reset    chan []string
	resetAll chan struct{}
	quit     chan struct{}
}

// ErrTimeout is returned by Wait() when failed to receive all the signals in time.
var ErrTimeout = errors.New("timeout")

// New creates and initializes a new SyncBus. It uses a shared timeout for all the Wait calls.
func New(timeout time.Duration) *SyncBus {
	b := &SyncBus{
		timeout:  timeout,
		signals:  make(map[string]bool),
		wait:     make(chan waitItem),
		signal:   make(chan []string),
		reset:    make(chan []string),
		resetAll: make(chan struct{}),
		quit:     make(chan struct{}),
	}

	go b.run()
	return b
}

func (b *SyncBus) nextTimeout(now time.Time) <-chan time.Time {
	if len(b.waiting) == 0 {
		return nil
	}

	to := b.waiting[0].deadline.Sub(time.Now())
	return time.After(to)
}

func (b *SyncBus) addWaiting(now time.Time, w waitItem) {
	w.deadline = now.Add(b.timeout)
	b.waiting = append(b.waiting, w)
}

func (b *SyncBus) setSignal(keys []string) {
	for _, key := range keys {
		b.signals[key] = true
	}
}

func (b *SyncBus) timeoutWaiting(now time.Time) {
	for i, w := range b.waiting {
		if w.deadline.After(now) {
			b.waiting = b.waiting[i:]
			return
		}

		w.signal <- ErrTimeout
	}

	b.waiting = nil
}

func (b *SyncBus) signalWaiting(now time.Time) {
	var keep []waitItem
	for _, w := range b.waiting {
		var keepItem bool
		for _, key := range w.keys {
			if !b.signals[key] {
				keepItem = true
				break
			}
		}

		if keepItem {
			keep = append(keep, w)
			continue
		}

		w.signal <- nil
	}

	b.waiting = keep
}

func (b *SyncBus) resetSignals(keys []string) {
	for i := range keys {
		delete(b.signals, keys[i])
	}
}

func (b *SyncBus) resetAllSignals() {
	b.signals = make(map[string]bool)
}

func (b *SyncBus) run() {
	var to <-chan time.Time
	for {
		select {
		case <-to:
			now := time.Now()
			b.timeoutWaiting(now)
			to = b.nextTimeout(now)
		case wait := <-b.wait:
			now := time.Now()
			b.addWaiting(now, wait)
			b.signalWaiting(now)
			to = b.nextTimeout(now)
		case signal := <-b.signal:
			now := time.Now()
			b.setSignal(signal)
			b.signalWaiting(now)
			to = b.nextTimeout(now)
		case reset := <-b.reset:
			b.resetSignals(reset)
		case <-b.resetAll:
			b.resetAllSignals()
		case <-b.quit:
			return
		}
	}
}

// Wait blocks until all the signals represented by the keys are set, or
// returns an ErrTimeout if the timeout, counted from the call to Wait,
// expires.
//
// It returns only ErrTimeout or nil.
//
// If the receiver *SyncBus is nil, or no key argument is passed to it,
// it is a noop.
func (b *SyncBus) Wait(keys ...string) error {
	if b == nil || len(keys) == 0 {
		return nil
	}

	w := waitItem{
		keys:   keys,
		signal: make(chan error, 1),
	}

	b.wait <- w
	err := <-w.signal
	return err
}

// Signal sets one or more signals represented by the keys.
//
// If the receiver *SyncBus is nil, or no key argument is passed to it,
// it is a noop.
func (b *SyncBus) Signal(keys ...string) {
	if b == nil || len(keys) == 0 {
		return
	}

	b.signal <- keys
}

// ResetSignals clears the set signals defined by the provided keys.
//
// If the receiver *SyncBus is nil, or no key argument is passed to it,
// it is a noop.
func (b *SyncBus) ResetSignals(keys ...string) {
	if b == nil || len(keys) == 0 {
		return
	}

	b.reset <- keys
}

// Reset clears all the signals.
//
// If the receiver *SyncBus is nil, it is a noop.
func (b *SyncBus) Reset() {
	if b == nil {
		return
	}

	b.resetAll <- struct{}{}
}

// Close tears down the SyncBus. If the receiver is nil, it is a noop.
func (b *SyncBus) Close() {
	if b == nil {
		return
	}

	close(b.quit)
}
