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
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, trailing at 250ms

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, trailing at 550ms

	// Output:
	// Hello, world!
	// Hello, world!
}

func ExampleNew_withLeading() {
	// Create a new debouncer that will call the callback function immediately
	// on the first call, and then wait 100 milliseconds since the last call
	// before calling the callback function again.
	debounced, _ := debounce.New(
		100*time.Millisecond,
		func() {
			fmt.Println("Hello, world!")
		},
		debounce.Leading(),
	)

	debounced()                       // leading trigger
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(250 * time.Millisecond) // +250ms = 400ms, wait expired at 350ms

	debounced()                       // leading trigger
	time.Sleep(75 * time.Millisecond) // +75ms = 475ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 550ms
	debounced()
	time.Sleep(250 * time.Millisecond) // +250ms = 800ms, wait expired at 750ms

	// Output:
	// Hello, world!
	// Hello, world!
}

func ExampleNew_withLeadingAndTrailing() {
	// Create a new debouncer that will call the callback function immediately
	// on the first call, and then wait 100 milliseconds since the last call
	// before calling the callback function again.
	debounced, _ := debounce.New(
		100*time.Millisecond,
		func() {
			fmt.Println("Hello, world!")
		},
		debounce.Leading(), debounce.Trailing(),
	)

	debounced()                       // leading trigger
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, trailing at 250ms

	// Output:
	// Hello, world!
	// Hello, world!
}

func ExampleNew_withReset() {
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

func ExampleNew_withMaxWait() {
	// Create a new debouncer that will wait 100 milliseconds since the last
	// call before calling the callback function. On repeated calls, it will
	// wait no more than 500 milliseconds before calling the callback function.
	debounced, _ := debounce.New(
		100*time.Millisecond,
		func() {
			fmt.Println("Hello, world!")
		},
		debounce.MaxWait(500*time.Millisecond),
	)

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 225ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 300ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 525ms, maxWait expired at 500ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 600ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 750ms, wait expired at 700ms

	// Output:
	// Hello, world!
	// Hello, world!
}

func ExampleNew_withMaxWaitAndLeading() {
	// Create a new debouncer that will wait 100 milliseconds since the last
	// call before calling the callback function. On repeated calls, it will
	// wait no more than 500 milliseconds before calling the callback function.
	debounced, _ := debounce.New(
		100*time.Millisecond,
		func() {
			fmt.Println("Hello, world!")
		},
		debounce.MaxWait(500*time.Millisecond),
		debounce.Leading(),
	)

	debounced()                       // leading trigger
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 225ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 300ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 525ms, maxWait expired at 500ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 600ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 750ms, wait expired at 650ms

	// Output:
	// Hello, world!
	// Hello, world!
}

func ExampleNew_withMaxWaitLeadingAndTrailing() {
	// Create a new debouncer that will wait 100 milliseconds since the last
	// call before calling the callback function. On repeated calls, it will
	// wait no more than 500 milliseconds before calling the callback function.
	debounced, _ := debounce.New(
		100*time.Millisecond,
		func() {
			fmt.Println("Hello, world!")
		},
		debounce.MaxWait(500*time.Millisecond),
		debounce.Leading(),
		debounce.Trailing(),
	)

	debounced()                       // leading trigger
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 225ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 300ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 525ms, maxWait expired at 500ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 600ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 750ms, trailing at 650ms

	// Output:
	// Hello, world!
	// Hello, world!
	// Hello, world!
}

func ExampleNew_withMaxWaitAndReset() {
	// Create a new debouncer that will wait 100 milliseconds since the last
	// call before calling the callback function. On repeated calls, it will
	// wait no more than 500 milliseconds before calling the callback function.
	debounced, reset := debounce.New(
		100*time.Millisecond,
		func() {
			fmt.Println("Hello, world!")
		},
		debounce.MaxWait(500*time.Millisecond),
	)

	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 225ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 300ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	reset()
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, reset at 450ms
	debounced()
	time.Sleep(75 * time.Millisecond) // +75ms = 675ms
	debounced()
	time.Sleep(150 * time.Millisecond) // +150ms = 825ms, wait expired at 775ms

	// Output:
	// Hello, world!
}

func ExampleNewMutable() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions.
	debounced, _ := debounce.NewMutable(100 * time.Millisecond)

	debounced(func() { fmt.Println("Hello, world! #1") })
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced(func() { fmt.Println("Hello, world! #2") })
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced(func() { fmt.Println("Hello, world! #3") })
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, trailing at 250ms

	debounced(func() { fmt.Println("Hello, world! #4") })
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced(func() { fmt.Println("Hello, world! #5") })
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	debounced(func() { fmt.Println("Hello, world! #6") })
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, trailing at 550ms

	// Output:
	// Hello, world! #3
	// Hello, world! #6
}

func ExampleNewMutable_withLeading() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions.
	debounced, _ := debounce.NewMutable(
		100*time.Millisecond,
		debounce.Leading(),
	)

	debounced(func() { fmt.Println("Hello, world! #1") }) // leading trigger
	time.Sleep(75 * time.Millisecond)                     // +75ms = 75ms
	debounced(func() { fmt.Println("Hello, world! #2") })
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced(func() { fmt.Println("Hello, world! #3") })
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, wait expired at 250ms

	debounced(func() { fmt.Println("Hello, world! #4") }) // leading trigger
	time.Sleep(75 * time.Millisecond)                     // +75ms = 375ms
	debounced(func() { fmt.Println("Hello, world! #5") })
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	debounced(func() { fmt.Println("Hello, world! #6") })
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, wait expired at 550ms

	// Output:
	// Hello, world! #1
	// Hello, world! #4
}

func ExampleNewMutable_withLeadingAndTrailing() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions.
	debounced, _ := debounce.NewMutable(
		100*time.Millisecond,
		debounce.Leading(),
		debounce.Trailing(),
	)

	debounced(func() { fmt.Println("Hello, world! #1") }) // leading trigger
	time.Sleep(75 * time.Millisecond)                     // +75ms = 75ms
	debounced(func() { fmt.Println("Hello, world! #2") })
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced(func() { fmt.Println("Hello, world! #3") })
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, trailing at 250ms

	// Output:
	// Hello, world! #1
	// Hello, world! #3
}

func ExampleNewMutable_withReset() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions.
	debounced, reset := debounce.NewMutable(100 * time.Millisecond)

	debounced(func() { fmt.Println("Hello, world! #1") })
	time.Sleep(75 * time.Millisecond) // +75ms = 75ms
	debounced(func() { fmt.Println("Hello, world! #2") })
	time.Sleep(75 * time.Millisecond) // +75ms = 150ms
	debounced(func() { fmt.Println("Hello, world! #3") })
	time.Sleep(150 * time.Millisecond) // +150ms = 300ms, trailing at 250ms

	debounced(func() { fmt.Println("Hello, world! #4") })
	time.Sleep(75 * time.Millisecond) // +75ms = 375ms
	debounced(func() { fmt.Println("Hello, world! #5") })
	time.Sleep(75 * time.Millisecond) // +75ms = 450ms
	reset()
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, reset at 450ms

	debounced(func() { fmt.Println("Hello, world! #6") })
	time.Sleep(75 * time.Millisecond) // +75ms = 675ms
	debounced(func() { fmt.Println("Hello, world! #7") })
	time.Sleep(75 * time.Millisecond) // +75ms = 750ms
	debounced(func() { fmt.Println("Hello, world! #8") })
	time.Sleep(150 * time.Millisecond) // +150ms = 900ms, trailing at 850ms

	// Output:
	// Hello, world! #3
	// Hello, world! #8
}

func ExampleNewMutable_withMaxWait() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions, on repeated debounce calls, it will wait no
	// more than 500 milliseconds before calling the callback function.
	debounced, _ := debounce.NewMutable(
		100*time.Millisecond,
		debounce.MaxWait(500*time.Millisecond),
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

func ExampleNewMutable_withMaxWaitAndLeading() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions, on repeated debounce calls, it will wait no
	// more than 500 milliseconds before calling the callback function.
	debounced, _ := debounce.NewMutable(
		100*time.Millisecond,
		debounce.MaxWait(500*time.Millisecond),
		debounce.Leading(),
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
	time.Sleep(75 * time.Millisecond) // +75ms = 525ms
	debounced(func() { fmt.Println("Hello, world! #8") })
	time.Sleep(75 * time.Millisecond) // +75ms = 600ms, maxWait expired at 575ms
	debounced(func() { fmt.Println("Hello, world! #9") })
	time.Sleep(150 * time.Millisecond) // +150ms = 750ms, wait expired at 700ms

	// Output:
	// Hello, world! #1
	// Hello, world! #8
}

func ExampleNewMutable_withMaxWaitLeadingAndTrailing() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions, on repeated debounce calls, it will wait no
	// more than 500 milliseconds before calling the callback function.
	debounced, _ := debounce.NewMutable(
		200*time.Millisecond,
		debounce.MaxWait(500*time.Millisecond),
		debounce.Leading(),
		debounce.Trailing(),
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
	time.Sleep(75 * time.Millisecond) // +75ms = 525ms
	debounced(func() { fmt.Println("Hello, world! #8") })
	time.Sleep(75 * time.Millisecond) // +75ms = 600ms, maxWait expired at 575ms
	debounced(func() { fmt.Println("Hello, world! #9") })
	time.Sleep(50 * time.Millisecond) // +50ms = 650ms
	debounced(func() { fmt.Println("Hello, world! #10") })
	time.Sleep(250 * time.Millisecond) // +250ms = 900ms, wait expired at 850ms

	// Output:
	// Hello, world! #1
	// Hello, world! #8
	// Hello, world! #10
}

func ExampleNewMutable_withMaxWaitAndReset() {
	// Create a new debouncer that will wait 100 milliseconds before calling
	// given callback functions, on repeated debounce calls, it will wait no
	// more than 500 milliseconds before calling the callback function.
	debounced, reset := debounce.NewMutable(
		100*time.Millisecond,
		debounce.MaxWait(500*time.Millisecond),
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
	reset()
	time.Sleep(150 * time.Millisecond) // +150ms = 600ms, reset at 450ms
	debounced(func() { fmt.Println("Hello, world! #7") })
	time.Sleep(75 * time.Millisecond) // +75ms = 600ms
	debounced(func() { fmt.Println("Hello, world! #8") })
	time.Sleep(150 * time.Millisecond) // +150ms = 750ms, wait expired at 700ms

	// Output:
	// Hello, world! #8
}
