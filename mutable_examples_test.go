package debounce_test

import (
	"fmt"
	"time"

	"github.com/romdo/go-debounce"
)

func ExampleNewMutable() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions.
	debounced, _ := debounce.NewMutable(100 * time.Millisecond)

	debounced(func() { fmt.Println("Hello, world! #1") })
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced(func() { fmt.Println("Hello, world! #2") })
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced(func() { fmt.Println("Hello, world! #3") })
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, wait expired at 250ms

	debounced(func() { fmt.Println("Hello, world! #4") })
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced(func() { fmt.Println("Hello, world! #5") })
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	debounced(func() { fmt.Println("Hello, world! #6") })
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, wait expired at 550ms

	// Output:
	// Hello, world! #3
	// Hello, world! #6
}

func ExampleNewMutable_with_cancel() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions.
	debounced, cancel := debounce.NewMutable(100 * time.Millisecond)

	debounced(func() { fmt.Println("Hello, world! #1") })
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced(func() { fmt.Println("Hello, world! #2") })
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced(func() { fmt.Println("Hello, world! #3") })
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, wait expired at 250ms

	debounced(func() { fmt.Println("Hello, world! #4") })
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced(func() { fmt.Println("Hello, world! #5") })
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	cancel()
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, canceled at 450ms

	debounced(func() { fmt.Println("Hello, world! #6") })
	time.Sleep(75 * time.Millisecond) // +75ms = 675ms
	debounced(func() { fmt.Println("Hello, world! #7") })
	time.Sleep(75 * time.Millisecond) // +75ms = 750ms
	debounced(func() { fmt.Println("Hello, world! #8") })
	time.Sleep(150 * time.Millisecond) // +150ms = 900ms, wait expired at 850ms

	// Output:
	// Hello, world! #3
	// Hello, world! #8
}

func ExampleNewMutableWithMaxWait() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions, on repeated debounce calls, it will wait no
	// more than 500 milliseconds before calling the callback function.
	debounced, _ := debounce.NewMutableWithMaxWait(
		100*time.Millisecond, 500*time.Millisecond,
	)

	debounced(func() { fmt.Println("Hello, world! #1") })
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced(func() { fmt.Println("Hello, world! #2") })
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced(func() { fmt.Println("Hello, world! #3") })
	time.Sleep(75 * time.Millisecond) // +75ms = 225ms
	debounced(func() { fmt.Println("Hello, world! #4") })
	time.Sleep(75 * time.Millisecond) // +75ms = 300ms
	debounced(func() { fmt.Println("Hello, world! #5") })
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced(func() { fmt.Println("Hello, world! #6") })
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	debounced(func() { fmt.Println("Hello, world! #7") })
	time.Sleep(75 * time.Millisecond) // +75ms = 525ms, maxWait expired at 500ms
	debounced(func() { fmt.Println("Hello, world! #8") })
	time.Sleep(75 * time.Millisecond) // +75ms = 600ms
	debounced(func() { fmt.Println("Hello, world! #9") })
	time.Sleep(150 * time.Millisecond) // +150ms = 750ms, wait expired at 700ms

	// Output:
	// Hello, world! #7
	// Hello, world! #9
}

func ExampleNewMutableWithMaxWait_with_cancel() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions, on repeated debounce calls, it will wait no
	// more than 500 milliseconds before calling the callback function.
	debounced, cancel := debounce.NewMutableWithMaxWait(
		100*time.Millisecond, 500*time.Millisecond,
	)

	debounced(func() { fmt.Println("Hello, world! #1") })
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced(func() { fmt.Println("Hello, world! #2") })
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced(func() { fmt.Println("Hello, world! #3") })
	time.Sleep(75 * time.Millisecond) // +75ms = 225ms
	debounced(func() { fmt.Println("Hello, world! #4") })
	time.Sleep(75 * time.Millisecond) // +75ms = 300ms
	debounced(func() { fmt.Println("Hello, world! #5") })
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced(func() { fmt.Println("Hello, world! #6") })
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	cancel()
	time.Sleep(75 * time.Millisecond) // +150 = 525ms, canceled at 450ms
	debounced(func() { fmt.Println("Hello, world! #8") })
	time.Sleep(75 * time.Millisecond) // +75ms = 600ms
	debounced(func() { fmt.Println("Hello, world! #9") })
	time.Sleep(150 * time.Millisecond) // +150ms = 750ms, wait expired at 700ms

	// Output:
	// Hello, world! #9
}
