package debounce

import (
	"sync"
	"time"
)

// Debouncer provides debouncing functionality for function calls.
// It combines configuration and state into a single struct with methods
// for invoking and resetting the debounced function.
type Debouncer struct {
	// Configuration
	wait     time.Duration
	fn       func()
	leading  bool
	trailing bool
	maxWait  time.Duration

	// State
	mux        sync.Mutex
	dirty      bool
	firstCall  time.Time
	lastCall   time.Time
	lastInvoke time.Time
	timer      *time.Timer
}

// NewDebouncer creates a new Debouncer instance with the given wait duration,
// function, and options.
func NewDebouncer(
	wait time.Duration,
	f func(),
	opts ...Option,
) *Debouncer {
	d := &Debouncer{wait: wait}
	for _, opt := range opts {
		opt(d)
	}

	// If neither leading nor trailing is set, default to trailing.
	if !d.leading && !d.trailing {
		d.trailing = true
	}

	// If maxWait is less than wait, disable maxWait.
	if d.maxWait <= d.wait {
		d.maxWait = 0
	}

	if f != nil {
		d.fn = f
	}

	d.timer = stoppedTimer(d.callback)

	return d
}

// Debounce invokes the debounced function according to the configured options.
// This method is safe for concurrent use.
func (d *Debouncer) Debounce() {
	d.DebounceWith(nil)
}

// DebounceWith allows setting a new function to be debounced and invoking it
// according to the configured options. On repeated calls, the last passed
// function wins and is executed. This method is safe for concurrent use.
//
// If f is nil, the debounced function is not modified from its current value.
func (d *Debouncer) DebounceWith(f func()) {
	d.mux.Lock()
	defer d.mux.Unlock()

	if f != nil {
		d.fn = f
	}

	now := time.Now()

	if d.wait <= 0 {
		d.invoke(now)
		return
	}

	if d.shouldInvoke(now) {
		d.invoke(now)
	} else if d.trailing {
		d.timer.Reset(d.wait)
		d.dirty = true
	}

	d.lastCall = now
}

func (d *Debouncer) shouldInvoke(now time.Time) bool {
	sinceLastCall := now.Sub(d.lastCall)
	sinceLastInvoke := now.Sub(d.lastInvoke)
	sinceMaxWaitOrigin := now.Sub(d.maxWaitOrigin(now))

	exceededWait := d.lastCall.IsZero() ||
		sinceLastCall < 0 || sinceLastInvoke < 0 ||
		(sinceLastCall >= d.wait && sinceLastInvoke >= d.wait)
	exceededMaxWait := d.maxWait > 0 &&
		sinceMaxWaitOrigin >= d.maxWait

	return (d.leading && exceededWait) || exceededMaxWait
}

// maxWaitOrigin returns the time of the first call or the last invocation. This
// is used to determine if the maxWait has been exceeded.
func (d *Debouncer) maxWaitOrigin(now time.Time) time.Time {
	if d.firstCall.IsZero() {
		d.firstCall = now
	}

	if d.lastInvoke.IsZero() {
		return d.firstCall
	}

	return d.lastInvoke
}

// Reset resets the debouncer, discarding any pending invocation.
// This method is safe for concurrent use.
func (d *Debouncer) Reset() {
	d.mux.Lock()
	defer d.mux.Unlock()

	d.firstCall = time.Time{}
	d.lastCall = time.Time{}
	d.lastInvoke = time.Time{}
	d.clear()
}

// callback is called when timer expires.
func (d *Debouncer) callback() {
	d.mux.Lock()
	defer d.mux.Unlock()

	// This is extremely unlikely, but should be checked.
	if !d.dirty {
		return
	}

	now := time.Now()
	d.invoke(now)
}

// clear stops and clears any pending debounces, without resetting last call and
// invocation times. It should only be called while the mutex is already locked.
func (d *Debouncer) clear() {
	d.dirty = false
	d.timer.Stop()
}

// invoke executes the function and updates the last invoke time. It should only
// be called while the mutex is already locked.
func (d *Debouncer) invoke(now time.Time) {
	if f := d.fn; f != nil {
		d.lastInvoke = now
		go f()
	}
	d.clear()
}
