// Package debounce provides functions to debounce function calls, i.e., to
// ensure that a function is only executed after a certain amount of time has
// passed since the last call.
//
// Debouncing can be useful in scenarios where function calls may be triggered
// rapidly, such as in response to user input, but the underlying operation is
// expensive and only needs to be performed once per batch of calls.
package debounce

import (
	"sync"
	"time"
)

// New returns a debounced function that delays invoking f until after wait time
// has elapsed since the last time the debounced function was invoked.
//
// The returned cancel function can be used to cancel any pending invocation of
// f, but is not required to be called, so can be ignored if not needed.
//
// Both debounced and cancel functions are safe for concurrent use in
// goroutines, and can both be called multiple times.
//
// The debounced function does not wait for f to complete, so f needs to be
// thread-safe as it may be invoked again before the previous invocation
// completes.
func New(wait time.Duration, f func()) (debounced func(), cancel func()) {
	var mux sync.Mutex
	timer := stoppedTimer(f)

	debounced = func() {
		mux.Lock()
		defer mux.Unlock()

		timer.Reset(wait)
	}

	cancel = func() {
		mux.Lock()
		defer mux.Unlock()

		timer.Stop()
	}

	return debounced, cancel
}

// NewWithMaxWait returns a debounced function like New, but with a maximum wait
// time of maxWait, which is the maximum time f is allowed to be delayed before
// it is invoked.
//
// The returned cancel function can be used to cancel any pending invocation of
// f, but is not required to be called, so can be ignored if not needed.
//
// Both debounced and cancel functions are safe for concurrent use in
// goroutines, and can both be called multiple times.
//
// The debounced function does not wait for f to complete, so f needs to be
// thread-safe as it may be invoked again before the previous invocation
// completes.
func NewWithMaxWait(
	wait, maxWait time.Duration,
	f func(),
) (debounced func(), cancel func()) {
	var mux sync.Mutex
	var dirty bool
	var timer *time.Timer
	var maxTimer *time.Timer

	cb := func() {
		mux.Lock()
		defer mux.Unlock()

		if !dirty {
			return
		}

		go f()
		timer.Stop()
		maxTimer.Stop()
		dirty = false
	}

	timer = stoppedTimer(cb)
	maxTimer = stoppedTimer(cb)

	debounced = func() {
		mux.Lock()
		defer mux.Unlock()

		timer.Reset(wait)

		// Mark as dirty, and start maxTimer if we were not already dirty.
		if !dirty {
			dirty = true
			maxTimer.Reset(maxWait)
		}
	}

	cancel = func() {
		mux.Lock()
		defer mux.Unlock()

		timer.Stop()
		maxTimer.Stop()
		dirty = false
	}

	return debounced, cancel
}
