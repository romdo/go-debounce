// Package debounce provides functions to debounce function calls, i.e., to
// ensure that a function is only executed after a certain amount of time has
// passed since the last call.
//
// Debouncing can be useful in scenarios where function calls may be triggered
// rapidly, such as in response to user input, but the underlying operation is
// expensive and only needs to be performed once per batch of calls.
package debounce

import (
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
// If wait is zero or negative, the original function is returned as the
// debounced function, and the reset function is a no-op.
//
// If no options are provided, Trailing() is used by default.
func New(
	wait time.Duration,
	f func(),
	opts ...Option,
) (debounced func(), reset func()) {
	d := NewDebouncer(wait, f, opts...)

	return d.Debounce, d.Reset
}
