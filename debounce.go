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
// The returned cancel function can be used to cancel any pending invocation of
// f, but is not required to be called, so can be ignored if not needed.
//
// Both debounced and cancel functions are safe for concurrent use in
// goroutines, and can both be called multiple times.
//
// The debounced function does not wait for f to complete, so f needs to be
// thread-safe as it may be invoked again before the previous invocation
// completes.
func New(
	wait time.Duration,
	f func(),
	opts ...Option,
) (debounced func(), cancel func()) {
	conf := &Config{}
	conf.Set(opts...)

	return conf.New(wait, f)
}
