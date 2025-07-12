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

// NewMutable returns a debounced function that allows changing the debounced
// function on each call. The returned function has the signature func(f func())
// where f is the function to be debounced.
//
// On repeated calls, the last passed function wins and is executed. If the
// passed function is nil, the debounced function is not modified from its
// current value.
//
// This is useful when you need to debounce different functions based on
// runtime conditions, and you want the most recent function to be executed
// when the debounce period expires.
//
// The returned reset function can be used to reset the debounce, making it
// operate as if it had never been called. Any pending invocation will be
// discarded when reset is called.
//
// Both returned functions are safe for concurrent use in goroutines.
//
// If wait is zero or negative, each passed function is executed immediately
// without debouncing.
//
// If no options are provided, Trailing() is used by default.
func NewMutable(
	wait time.Duration,
	opts ...Option,
) (debounced func(f func()), reset func()) {
	d := NewDebouncer(wait, nil, opts...)

	return d.DebounceWith, d.Reset
}
