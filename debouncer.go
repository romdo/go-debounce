package debounce

import (
	"fmt"
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
		d.fn = f
	}

	d.timer = stoppedTimer(func() {
		fmt.Println("trigger timer")
		d.callback()
	})
	d.maxTimer = stoppedTimer(func() {
		fmt.Println("trigger maxTimer")
		d.callback()
	})

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

	if d.wait <= 0 {
		d.invoke(time.Now())
		return
	}

	now := time.Now()
	invokedLeading := false

	if d.leading {
		elapsed := now.Sub(d.lastCall)
		elapsedInvoke := now.Sub(d.lastInvoke)

		exceededWait := d.lastCall.IsZero() ||
			elapsed < 0 ||
			elapsedInvoke < 0 ||
			(elapsed >= d.wait && elapsedInvoke >= d.wait)

		if exceededWait {
			fmt.Println("trigger invoke from leading")
			d.invoke(now)
			invokedLeading = true
		}
	}

	if !invokedLeading {
		if d.trailing {
			d.timer.Reset(d.wait)
		}

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

	d.lastCall = time.Time{}
	d.lastInvoke = time.Time{}
	d.clear()
}

// callback is called when timer or maxTimer expires.
func (d *Debouncer) callback() {
	d.mux.Lock()
	defer d.mux.Unlock()

	if !d.dirty {
		return
	}

	now := time.Now()
	fmt.Println("trigger invoke from callback")
	d.invoke(now)
	d.clear()
}

// clear stops and clears any pending debounces, without resetting last call and
// invocation times. It should only be called while the mutex is already locked.
func (d *Debouncer) clear() {
	d.dirty = false
	d.maxTimer.Stop()
	d.timer.Stop()
}

// invoke executes the function and updates the last invoke time. It should only
// be called while the mutex is already locked.
func (d *Debouncer) invoke(now time.Time) {
	if f := d.fn; f != nil {
		d.lastInvoke = now
		go f()
	}
}
