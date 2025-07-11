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
// The returned reset function can be used to reset the debounce, making it
// operate as if it had never been called. Any pending invocation of f will be
// discarded when reset is called.
//
// Both debounced and reset functions are safe for concurrent use in
// goroutines, and can both be called multiple times.
//
// The debounced function does not wait for f to complete, so f needs to be
// concurrency-safe as it may be invoked again before the previous invocation
// returns.
//
// If no options are provided, WithTrailing is used by default.
func New(
	wait time.Duration,
	f func(),
	opts ...Option,
) (debounced func(), reset func()) {
	conf := &config{}
	for _, opt := range opts {
		opt(conf)
	}

	return debounce(wait, *conf, f)
}

type config struct {
	leading  bool
	trailing bool
	maxWait  time.Duration
}

type state struct {
	mux        sync.Mutex
	dirty      bool
	lastCall   time.Time
	lastInvoke time.Time
	timer      *time.Timer
	maxTimer   *time.Timer
}

// New creates a new debounced function that will invoke the given function
// after a delay. The debounced function will return a reset function that
// can be used to reset the debounce.
func debounce(
	wait time.Duration,
	conf config,
	f func(),
) (debounced func(), reset func()) {
	if wait <= 0 {
		return f, func() {}
	}

	// If neither leading nor trailing is set, default to trailing.
	if !conf.leading && !conf.trailing {
		conf.trailing = true
	}

	s := state{}

	invoke := func(now time.Time) {
		s.lastInvoke = now
		go f()
	}

	invokeLeading := func(now time.Time) bool {
		if !conf.leading {
			return false
		}

		elapsed := now.Sub(s.lastCall)
		elapsedInvoke := now.Sub(s.lastInvoke)
		exceededWait := s.lastCall.IsZero() ||
			elapsed < 0 || elapsedInvoke < 0 ||
			(elapsed >= wait && elapsedInvoke >= wait)
		exceededMaxWait := conf.maxWait > 0 && elapsedInvoke >= conf.maxWait

		if exceededWait || exceededMaxWait {
			invoke(now)

			return true
		}

		return false
	}

	cb := func() {
		s.mux.Lock()
		defer s.mux.Unlock()

		if !s.dirty {
			return
		}

		now := time.Now()

		invoke(now)
		s.timer.Stop()
		s.maxTimer.Stop()
		s.dirty = false
	}

	s.timer = stoppedTimer(cb)
	s.maxTimer = stoppedTimer(cb)

	debounced = func() {
		s.mux.Lock()
		defer s.mux.Unlock()

		now := time.Now()
		invokedLeading := invokeLeading(now)

		if !invokedLeading && conf.trailing {
			s.timer.Reset(wait)

			if conf.maxWait > 0 && !s.dirty {
				s.maxTimer.Reset(conf.maxWait)
			}
			s.dirty = true
		}

		s.lastCall = now
	}

	reset = func() {
		s.mux.Lock()
		defer s.mux.Unlock()

		s.timer.Stop()
		s.maxTimer.Stop()
		s.dirty = false
		s.lastInvoke = time.Time{}
		s.lastCall = time.Time{}
	}

	return debounced, reset
}
