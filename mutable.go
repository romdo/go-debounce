package debounce

import (
	"sync"
	"time"
)

// NewMutable returns a debounced function like New, but it allows callback
// function f to be changed, as a new callback function is passed to each
// invocation of the debounced function.
//
// The returned cancel function can be used to cancel any pending invocation of
// f, but is not required to be called, so can be ignored if not needed.
//
// Only the very last f passed to the debounced function is called when the
// delay expires and the callback function is invoked. Previous f values are
// discarded.
//
// Both debounced and cancel functions are safe for concurrent use in
// goroutines, and can both be called multiple times.
func NewMutable(wait time.Duration) (debounced func(f func()), cancel func()) {
	var mux sync.Mutex
	var fn func()

	timer := stoppedTimer(func() {
		mux.Lock()
		defer mux.Unlock()

		go fn()
	})

	debounced = func(f func()) {
		mux.Lock()
		defer mux.Unlock()

		fn = f
		timer.Reset(wait)
	}

	cancel = func() {
		mux.Lock()
		defer mux.Unlock()

		timer.Stop()
	}

	return debounced, cancel
}

// NewMutableWithMaxWait is a combination of NewMutable and NewWithMaxWait.
//
// When either of the wait or maxWait timers expire, the last f passed to the
// debounced function is called.
//
// The returned cancel function can be used to cancel any pending invocation of
// f, but is not required to be called, so can be ignored if not needed.
//
// Both debounced and cancel functions are safe for concurrent use in
// goroutines, and can both be called multiple times.
func NewMutableWithMaxWait(
	wait, maxWait time.Duration,
) (debounced func(f func()), cancel func()) {
	var mux sync.Mutex
	var fn func()
	var timer *time.Timer
	var maxTimer *time.Timer

	cb := func() {
		mux.Lock()
		defer mux.Unlock()

		if fn == nil {
			return
		}

		go fn()
		timer.Stop()
		maxTimer.Stop()
		fn = nil
	}

	timer = stoppedTimer(cb)
	maxTimer = stoppedTimer(cb)

	debounced = func(f func()) {
		mux.Lock()
		defer mux.Unlock()

		timer.Reset(wait)

		if fn == nil {
			maxTimer.Reset(maxWait)
		}

		fn = f
	}

	cancel = func() {
		mux.Lock()
		defer mux.Unlock()

		timer.Stop()
		maxTimer.Stop()
		fn = nil
	}

	return debounced, cancel
}
