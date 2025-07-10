package debounce

import (
	"sync"
	"time"
)

// Config is a configuration used for creating a debounced function.
type Config struct {
	// Leading returns an option that will cause the debounced function to invoke
	// the given function immediately, and then wait for the given duration before
	// invoking the function again.
	//
	// When only leading is used, a burst of calls immediately invokes the function,
	// any subsequent calls will be ignored until the wait duration has passed.
	Leading bool

	// Trailing returns an option that will cause the debounced function to be
	// invoked after the wait duration has passed since call or last invocation.
	//
	// When only trailing is used, a burst of calls will not invoke the function
	// until the wait duration has passed.
	//
	// If both Leading and Trailing are used, a burst of calls immediately invokes
	// the function, followed by another invocation after the wait duration has
	// passed since the last call. If only a single call is made, only one
	// invocation will occur. If two calls happens within the wait duration, the
	// function will be invoked twice.
	Trailing bool

	// MaxWait returns an option that will cause the debounced function to be
	// invoked every maxWait duration, even if the function is called repeatedly
	// within the wait duration.
	//
	// Without a max wait, the debounced function might never be invoked if the it
	// is called repeatedly within the wait duration.
	//
	// For example, if the wait duration is 100ms and the max wait duration is
	// 500ms, the debounced function will be invoked every 500ms, even if the
	// function is called non-stop every 10ms.
	MaxWait time.Duration
}

// Set sets the options for the debounced function with Option functions
func (c *Config) Set(o ...Option) {
	for _, opt := range o {
		opt(c)
	}
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
func (c *Config) New(
	wait time.Duration,
	f func(),
) (debounced func(), reset func()) {
	if wait <= 0 {
		return f, func() {}
	}

	if c == nil {
		c = &Config{Trailing: true}
	}

	// Create a copy of the config so that it can be modified without affecting
	// existing debounce functions.
	conf := *c

	// If neither leading nor trailing is set, default to trailing.
	if !conf.Leading && !conf.Trailing {
		conf.Trailing = true
	}

	s := state{}

	invoke := func(now time.Time) {
		s.lastInvoke = now
		go f()
	}

	invokeLeading := func(now time.Time) bool {
		if !conf.Leading {
			return false
		}

		elapsed := now.Sub(s.lastCall)
		elapsedInvoke := now.Sub(s.lastInvoke)
		exceededWait := s.lastCall.IsZero() ||
			elapsed < 0 || elapsedInvoke < 0 ||
			(elapsed >= wait && elapsedInvoke >= wait)
		exceededMaxWait := conf.MaxWait > 0 && elapsedInvoke >= conf.MaxWait

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

		if !invokedLeading && conf.Trailing {
			s.timer.Reset(wait)

			if conf.MaxWait > 0 && !s.dirty {
				s.maxTimer.Reset(conf.MaxWait)
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
