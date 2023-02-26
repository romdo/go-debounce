package debounce

import (
	"sync"
	"time"
)

type Config struct {
	Leading  bool
	Trailing bool
	MaxWait  time.Duration
}

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

func (c *Config) New(
	wait time.Duration,
	f func(),
) (debounced func(), cancel func()) {
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

		if !invokeLeading(now) && conf.Trailing {
			s.timer.Reset(wait)

			if conf.MaxWait > 0 && !s.dirty {
				s.maxTimer.Reset(conf.MaxWait)
			}
			s.dirty = true
		}

		s.lastCall = now
	}

	cancel = func() {
		s.mux.Lock()
		defer s.mux.Unlock()

		s.timer.Stop()
		s.maxTimer.Stop()
		s.dirty = false
	}

	return debounced, cancel
}
