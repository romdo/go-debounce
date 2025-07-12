package debounce

import (
	"sync"
	"sync/atomic"
	"time"
)

// Debouncer provides debouncing functionality for function calls.
// It combines configuration and state into a single struct with methods
// for invoking and resetting the debounced function.
type Debouncer struct {
	// Configuration
	wait     time.Duration
	leading  bool
	trailing bool
	maxWait  time.Duration

	// State
	fn         atomic.Pointer[func()]
	mux        sync.Mutex
	dirty      bool
	lastCall   time.Time
	lastInvoke time.Time
	maxTimer   *time.Timer
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
		d.fn.Store(&f)
	}

	d.timer = stoppedTimer(d.timerCallback)
	d.maxTimer = stoppedTimer(d.timerCallback)

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
	if f != nil {
		d.fn.Store(&f)
	}

	d.mux.Lock()
	defer d.mux.Unlock()

	if d.wait <= 0 {
		d.invoke(time.Now())
		return
	}

	now := time.Now()
	invokedLeading := d.invokeLeading(now)

	if !invokedLeading && d.trailing {
		d.timer.Reset(d.wait)

		if d.maxWait > 0 && !d.dirty {
			d.maxTimer.Reset(d.maxWait)
		}
		d.dirty = true
	}

	d.lastCall = now
}

// Reset resets the debouncer, discarding any pending invocation.
// This method is safe for concurrent use.
func (d *Debouncer) Reset() {
	d.mux.Lock()
	defer d.mux.Unlock()

	d.dirty = false
	d.lastCall = time.Time{}
	d.lastInvoke = time.Time{}
	d.maxTimer.Stop()
	d.timer.Stop()
}

// timerCallback is called when the timer expires.
func (d *Debouncer) timerCallback() {
	d.mux.Lock()
	defer d.mux.Unlock()

	if !d.dirty {
		return
	}

	now := time.Now()

	d.invoke(now)
	d.timer.Stop()
	d.maxTimer.Stop()
	d.dirty = false
}

// invoke executes the function and updates the last invoke time.
func (d *Debouncer) invoke(now time.Time) {
	if f := d.fn.Load(); f != nil && *f != nil {
		d.lastInvoke = now
		go (*f)()
	}
}

// invokeLeading handles leading edge invocation logic.
func (d *Debouncer) invokeLeading(now time.Time) bool {
	if !d.leading {
		return false
	}

	elapsed := now.Sub(d.lastCall)
	elapsedInvoke := now.Sub(d.lastInvoke)
	exceededWait := d.lastCall.IsZero() ||
		elapsed < 0 || elapsedInvoke < 0 ||
		(elapsed >= d.wait && elapsedInvoke >= d.wait)
	exceededMaxWait := d.maxWait > 0 && elapsedInvoke >= d.maxWait

	if exceededWait || exceededMaxWait {
		d.invoke(now)
		return true
	}

	return false
}
