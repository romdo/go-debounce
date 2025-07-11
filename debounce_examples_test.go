package debounce_test

import (
	"fmt"
	"time"

	"github.com/romdo/go-debounce"
)

func ExampleNew() {
	// Create a new debouncer that will wait 100 milliseconds since the last
	// call before calling the callback function.
	debounced, _ := debounce.New(100*time.Millisecond, func() {
		fmt.Println("Hello, world!")
	})

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, wait expired at 250ms

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 675ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 750ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 900ms, wait expired at 850ms

	// Output:
	// Hello, world!
	// Hello, world!
}

func ExampleNew_with_reset() {
	// Create a new debouncer that will wait 100 milliseconds since the last
	// call before calling the callback function.
	debounced, reset := debounce.New(100*time.Millisecond, func() {
		fmt.Println("Hello, world!")
	})

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, wait expired at 250ms

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	reset()
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, reset at 450ms

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 675ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 750ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 900ms, wait expired at 850ms

	// Output:
	// Hello, world!
	// Hello, world!
}
