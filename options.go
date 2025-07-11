package debounce

import (
	"time"
)

// Option is a function that can be used to configure the debounced function.
type Option func(*config)

// WithLeading returns an option that will cause the debounced function to
// invoke the given function immediately, and then wait for the given duration
// before invoking the function again.
//
// When only leading is used, a burst of calls immediately invokes the function,
// any subsequent calls will be ignored until the wait duration has passed.
func WithLeading() Option {
	return func(c *config) {
		c.leading = true
	}
}

// WithTrailing returns an option that will cause the debounced function to be
// invoked after the wait duration has passed since call or last invocation.
//
// When only trailing is used, a burst of calls will not invoke the function
// until the wait duration has passed.
//
// If both Leading and WithTrailing are used, a burst of calls immediately
// invokes the function, followed by another invocation after the wait duration
// has passed since the last call. If only a single call is made, only one
// invocation will occur. If two calls happens within the wait duration, the
// function will be invoked twice.
func WithTrailing() Option {
	return func(c *config) {
		c.trailing = true
	}
}

// WithMaxWait returns an option that will cause the debounced function to be
// invoked every maxWait duration, even if the function is called repeatedly
// within the wait duration.
//
// Without a max wait, the debounced function might never be invoked if the it
// is called repeatedly within the wait duration.
//
// For example, if the wait duration is 100ms and the max wait duration is
// 500ms, the debounced function will be invoked every 500ms, even if the
// function is called non-stop every 10ms.
func WithMaxWait(maxWait time.Duration) Option {
	return func(c *config) {
		c.maxWait = maxWait
	}
}
