package debounce

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var maxRetries = flag.Int("max-retries", 0, "Maximum number of retries")

// Due to the timing-based nature of the test suite, we want to support
// automatically retrying the tests a few times to avoid flakiness.
func TestMain(m *testing.M) {
	flag.Parse()

	code := m.Run()

	for i := 0; code != 0 && i < *maxRetries; i++ {
		fmt.Fprintf(os.Stderr,
			"===\n=== WARN  Tests failed, retrying (%d/%d)...\n===\n",
			i+1, *maxRetries,
		)
		code = m.Run()
	}

	os.Exit(code)
}

type testCase struct {
	name               string
	wait               time.Duration
	options            []Option
	actions            map[int64]testAction
	legacyCalls        []testOp
	calls              []int64
	resets             []int64
	wantInvocations    []int64
	assertMargin       int64
	legacyWantTriggers map[time.Duration]int64
}

type testOp struct {
	delay time.Duration
	reset bool
}

type testAction struct {
	call       bool
	reset      bool
	wantInvocs int64
}

func runTestCases(t *testing.T, tests []testCase) {
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var startTime int64
			var invocations []int64
			mux := sync.Mutex{}

			var n int64
			f := func() {
				mux.Lock()
				defer mux.Unlock()

				offset := time.Now().UnixMilli() - startTime
				invocations = append(invocations, offset)
				atomic.AddInt64(&n, 1)
			}
			debouncedFunc, resetFunc := New(tt.wait, f, tt.options...)
			var lastWantInvocs int64

			wg := sync.WaitGroup{}

			if len(tt.calls) > 0 {
				startTime = time.Now().UnixMilli()
				for _, x := range tt.calls {
					wg.Add(1)
					go func(x int64) {
						defer wg.Done()
						time.Sleep(time.Duration(x) * time.Millisecond)
						debouncedFunc()
					}(x)
				}

				for _, x := range tt.resets {
					wg.Add(1)
					go func(x int64) {
						defer wg.Done()
						time.Sleep(time.Duration(x) * time.Millisecond)
						resetFunc()
					}(x)
				}
			} else if len(tt.actions) > 0 {
				startTime = time.Now().UnixMilli()
				for ms, action := range tt.actions {
					wg.Add(1)
					dur := time.Duration(ms) * time.Millisecond

					go func(offset time.Duration, act testAction) {
						defer wg.Done()
						time.Sleep(offset)
						if act.call {
							debouncedFunc()
						} else if act.reset {
							resetFunc()
						} else {
							atomic.StoreInt64(&lastWantInvocs, act.wantInvocs)
							got := atomic.LoadInt64(&n)
							assert.Equal(t, act.wantInvocs, got, "at %s", dur)
						}
					}(dur, action)
				}
			} else {
				for _, op := range tt.legacyCalls {
					wg.Add(1)
					go func(delay time.Duration, reset bool) {
						defer wg.Done()
						time.Sleep(delay)
						if reset {
							resetFunc()
						} else {
							debouncedFunc()
						}
					}(op.delay, op.reset)
				}

				for delay, count := range tt.legacyWantTriggers {
					wg.Add(1)
					go func(interval time.Duration, count int64) {
						defer wg.Done()
						time.Sleep(interval)

						got := atomic.LoadInt64(&n)
						assert.Equal(t, count, got, "at %s", interval)
					}(delay, count)
				}
			}

			wg.Wait()

			if len(tt.calls) > 0 {
				// Wait a bit of extra time just to try and make sure there's
				// no lingering debounce left.
				time.Sleep(tt.wait * 2)

				assert.Len(t, invocations, len(tt.wantInvocations), "invocations")
				fmt.Printf("invocations: %v\n", invocations)

				margin := tt.assertMargin
				if margin == 0 {
					margin = 30
				}

				for _, want := range tt.wantInvocations {
					found := -1
					for i, inv := range invocations {
						// If the invocation is within 30ms of the want, we've
						// found it.
						if want-margin < inv && want+margin > inv {
							found = i
							break
						}
					}
					assert.NotEqual(t, -1, found, "at %d ms", want)
					// Remove the invocation from the list.
					if found != -1 {
						invocations = append(invocations[:found], invocations[found+1:]...)
					}
				}
				assert.Equal(t, 0, len(invocations), "invocations left")
			}

			if len(tt.actions) > 0 {
				// Wait a bit of extra time just to try and make sure there's
				// no lingering debounce left.
				time.Sleep(tt.wait * 2)
				assert.Equal(t, lastWantInvocs, n, "last want invocations")
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name: "one call, one trigger",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				250: {wantInvocs: 0},
				350: {wantInvocs: 1}, // trailing trigger at 300ms
			},
		},
		{
			name: "two calls, two triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				250: {wantInvocs: 0},
				350: {wantInvocs: 1}, // trailing trigger at 300ms

				400: {call: true},
				550: {wantInvocs: 1},
				650: {wantInvocs: 2}, // trailing trigger at 600ms
			},
		},
		{
			name: "one burst of calls, one triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				250: {call: true},
				300: {call: true},
				350: {call: true},
				400: {call: true},
				450: {call: true},
				500: {call: true},
				650: {wantInvocs: 0},
				750: {wantInvocs: 1}, // trailing trigger at 700ms
			},
		},
		{
			name: "one burst of calls with a reset, one trigger",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				250: {call: true},
				300: {call: true},
				350: {call: true},
				400: {call: true},
				450: {reset: true},

				500: {call: true},
				550: {call: true},
				600: {call: true},
				750: {wantInvocs: 0},
				850: {wantInvocs: 1}, // trailing trigger at 800ms
			},
		},
		{
			name: "two close bursts of calls, two triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				350: {wantInvocs: 0},
				450: {wantInvocs: 1}, // trailing trigger at 400ms

				500: {call: true},
				550: {call: true},
				600: {call: true},
				750: {wantInvocs: 1},
				850: {wantInvocs: 2}, // trailing trigger at 800ms
			},
		},
		{
			name: "two close bursts of calls, longer wait, one trigger",
			wait: 400 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},

				500: {call: true},
				550: {call: true},
				600: {call: true},

				950:  {wantInvocs: 0},
				1050: {wantInvocs: 1}, // trailing trigger at 1000ms
			},
		},
		{
			name: "two bursts of calls, two triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				250: {call: true},
				300: {call: true},
				450: {wantInvocs: 0},
				550: {wantInvocs: 1}, // trailing trigger at 500ms

				800:  {call: true},
				850:  {call: true},
				900:  {call: true},
				950:  {call: true},
				1000: {call: true},
				1150: {wantInvocs: 1},
				1250: {wantInvocs: 2}, // trailing trigger at 1200ms
			},
		},
		{
			name: "two close bursts of calls, reset, two triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				350: {wantInvocs: 0},
				450: {wantInvocs: 1}, // trailing trigger at 300ms
				451: {reset: true},

				500: {call: true},
				550: {call: true},
				600: {call: true},
				750: {wantInvocs: 1},
				850: {wantInvocs: 2}, // trailing trigger at 800ms
			},
		},
	}
	runTestCases(t, tests)
}

func TestNew_with_Trailing(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name: "one call, one trigger",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				250: {wantInvocs: 0},
				350: {wantInvocs: 1}, // trailing trigger at 300ms
			},
		},
		{
			name: "two calls, two triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				250: {wantInvocs: 0},
				350: {wantInvocs: 1}, // trailing trigger at 300ms

				400: {call: true},
				550: {wantInvocs: 1},
				650: {wantInvocs: 2}, // trailing trigger at 600ms
			},
		},
		{
			name: "one burst of calls, one triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				250: {call: true},
				300: {call: true},
				350: {call: true},
				400: {call: true},
				450: {call: true},
				500: {call: true},
				650: {wantInvocs: 0},
				750: {wantInvocs: 1}, // trailing trigger at 700ms
			},
		},
		{
			name: "one burst of calls with a reset, one trigger",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				250: {call: true},
				300: {call: true},
				350: {call: true},
				400: {call: true},
				450: {reset: true},

				500: {call: true},
				550: {call: true},
				600: {call: true},
				750: {wantInvocs: 0},
				850: {wantInvocs: 1}, // trailing trigger at 800ms
			},
		},
		{
			name: "two close bursts of calls, two triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				350: {wantInvocs: 0},
				450: {wantInvocs: 1}, // trailing trigger at 400ms

				500: {call: true},
				550: {call: true},
				600: {call: true},
				750: {wantInvocs: 1},
				850: {wantInvocs: 2}, // trailing trigger at 800ms
			},
		},
		{
			name: "two close bursts of calls, longer wait, one trigger",
			wait: 400 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},

				500: {call: true},
				550: {call: true},
				600: {call: true},

				950:  {wantInvocs: 0},
				1050: {wantInvocs: 1}, // trailing trigger at 1000ms
			},
		},
		{
			name: "two bursts of calls, two triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				250: {call: true},
				300: {call: true},
				450: {wantInvocs: 0},
				550: {wantInvocs: 1}, // trailing trigger at 500ms

				800:  {call: true},
				850:  {call: true},
				900:  {call: true},
				950:  {call: true},
				1000: {call: true},
				1150: {wantInvocs: 1},
				1250: {wantInvocs: 2}, // trailing trigger at 1200ms
			},
		},
		{
			name: "two close bursts of calls, reset, two triggers",
			wait: 200 * time.Millisecond,
			actions: map[int64]testAction{
				100: {call: true},
				150: {call: true},
				200: {call: true},
				350: {wantInvocs: 0},
				450: {wantInvocs: 1}, // trailing trigger at 300ms
				451: {reset: true},

				500: {call: true},
				550: {call: true},
				600: {call: true},
				750: {wantInvocs: 1},
				850: {wantInvocs: 2}, // trailing trigger at 800ms
			},
		},
	}
	runTestCases(t, tests)
}

func TestNew_with_Leading(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:    "one call, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			actions: map[int64]testAction{
				50:  {wantInvocs: 0},
				100: {call: true},
				150: {wantInvocs: 1}, // leading trigger at 100ms
			},
		},
		{
			name:    "two calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			actions: map[int64]testAction{
				50:  {wantInvocs: 0},
				100: {call: true},
				150: {wantInvocs: 1}, // leading trigger at 100ms

				400: {call: true},
				450: {wantInvocs: 2}, // leading trigger at 400ms
			},
		},
		{
			name:    "one burst of calls, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			actions: map[int64]testAction{
				50:   {wantInvocs: 0},
				100:  {call: true},
				150:  {wantInvocs: 1}, // leading trigger at 100ms
				151:  {call: true},
				200:  {call: true},
				250:  {call: true},
				300:  {call: true},
				350:  {call: true},
				400:  {call: true},
				450:  {call: true},
				500:  {call: true},
				1000: {wantInvocs: 1},
			},
		},
		{
			name:    "one burst of calls with a reset, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			actions: map[int64]testAction{
				50:  {wantInvocs: 0},
				100: {call: true},
				150: {wantInvocs: 1}, // leading trigger at 100ms
				151: {call: true},
				200: {call: true},
				250: {call: true},
				300: {call: true},
				350: {reset: true},
				351: {wantInvocs: 1},
				400: {call: true},
				450: {wantInvocs: 2}, // leading trigger at 400ms
				451: {call: true},
				500: {call: true},
				550: {call: true},
			},
		},
		{
			name:    "two close bursts of calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			actions: map[int64]testAction{
				50:  {wantInvocs: 0},
				100: {call: true},
				150: {wantInvocs: 1}, // leading trigger at 100ms
				151: {call: true},
				200: {call: true},

				450: {wantInvocs: 1},
				500: {call: true},
				550: {wantInvocs: 2}, // leading trigger at 500ms
				551: {call: true},
				600: {call: true},
			},
		},
		{
			name:    "two close bursts of calls, longer wait, one trigger",
			wait:    500 * time.Millisecond,
			options: []Option{Leading()},
			actions: map[int64]testAction{
				50:  {wantInvocs: 0},
				100: {call: true},
				150: {wantInvocs: 1}, // leading trigger at 100ms
				151: {call: true},
				200: {call: true},

				500: {call: true},
				551: {call: true},
				600: {call: true},
			},
		},
		{
			name:    "two bursts of calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			actions: map[int64]testAction{
				50:  {wantInvocs: 0},
				100: {call: true},
				150: {wantInvocs: 1}, // leading trigger at 100ms
				151: {call: true},
				200: {call: true},
				250: {call: true},
				300: {call: true},

				550: {wantInvocs: 1},
				600: {call: true},
				650: {wantInvocs: 2}, // leading trigger at 600ms
				651: {call: true},
				700: {call: true},
				750: {call: true},
				800: {call: true},
			},
		},
		{
			name:    "two close bursts of calls, reset, two triggers",
			wait:    500 * time.Millisecond,
			options: []Option{Leading()},
			actions: map[int64]testAction{
				50:  {wantInvocs: 0},
				100: {call: true},
				150: {wantInvocs: 1}, // leading trigger at 100ms
				151: {call: true},
				200: {call: true},
				450: {reset: true},

				451: {wantInvocs: 1},
				500: {call: true},
				550: {wantInvocs: 2}, // leading trigger at 500ms
				551: {call: true},
				600: {call: true},
			},
		},
	}
	runTestCases(t, tests)
}

func TestNew_with_Leading_and_Trailing(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:    "one call, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100ms.
			},
		},
		{
			name:    "two calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 400,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100ms.
				400, // Leading trigger at 400ms.
			},
		},
		{
			name:    "one burst of calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 151, 200, 250, 300, 350, 400, 450, 500,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100ms.
				700, // Trailing trigger from call at 500ms.
			},
		},
		{
			name:    "one burst of calls with a reset, three triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 150, 200, 250, 300, 350, 400, 450, 500, 550, 600,
			},
			resets: []int64{
				475,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100ms.
				500, // Leading trigger at 500ms.
				800, // Trailing trigger from call at 600ms.
			},
		},
		{
			name:    "two close bursts of calls, three triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100ms.
				400, // Trailing trigger from call at 200ms.
				800, // Trailing trigger from call at 600ms.
			},
		},
		{
			name:    "two close bursts of calls, longer wait, two triggers",
			wait:    400 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			wantInvocations: []int64{
				100,  // Leading trigger at 100ms.
				1000, // Trailing trigger from call at 600ms.
			},
		},
		{
			name:    "two bursts of calls, four triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 150, 200, 250, 300,
				800, 850, 900, 950, 1000,
			},
			wantInvocations: []int64{
				100,  // Leading trigger at 100ms.
				500,  // Trailing trigger from call at 300ms.
				800,  // Leading trigger at 800ms.
				1200, // Trailing trigger from call at 1000ms.
			},
		},
		{
			name:    "two close bursts of calls, reset, four triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			resets: []int64{
				450,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100ms.
				400, // Trailing trigger from call at 200ms.
				500, // Leading trigger at 500ms.
				800, // Trailing trigger from call at 600ms.
			},
		},
	}

	runTestCases(t, tests)
}

func TestNew_with_MaxWait(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name: "all within wait time",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			legacyCalls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 50 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 150 * time.Millisecond},
				{delay: 150 * time.Millisecond},
			},
			legacyWantTriggers: map[time.Duration]int64{
				300 * time.Millisecond: 0,
				// tick over at 350ms (150ms + 200ms)
				400 * time.Millisecond: 1,
				// still 1 at at the end
				1000 * time.Millisecond: 1,
			},
		},
		{
			name: "until right before maxWait",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			legacyCalls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
			},
			legacyWantTriggers: map[time.Duration]int64{
				450 * time.Millisecond: 0,
				// tick over at 500ms via maxWait
				550 * time.Millisecond: 1,
				// still 1 at at the end
				1000 * time.Millisecond: 1,
			},
		},
		{
			name: "until right after maxWait",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			legacyCalls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 600 * time.Millisecond},
			},
			legacyWantTriggers: map[time.Duration]int64{
				450 * time.Millisecond: 0,
				// tick over at 500ms via maxWait
				550 * time.Millisecond: 1,
				750 * time.Millisecond: 1,
				// tick over at 800ms (600ms + 200ms)
				850 * time.Millisecond: 2,
				// still 2 at at the end
				1500 * time.Millisecond: 2,
			},
		},
		{
			name: "until two maxWaits and one wait exipry",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			legacyCalls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 490 * time.Millisecond},
				// maxWait triggers at 500ms (00ms + 500ms)
				{delay: 510 * time.Millisecond},
				{delay: 600 * time.Millisecond},
				{delay: 700 * time.Millisecond},
				{delay: 800 * time.Millisecond},
				{delay: 900 * time.Millisecond},
				{delay: 990 * time.Millisecond},
				// maxWait triggers at 1000ms (500ms + 500ms)
				{delay: 1010 * time.Millisecond},
				{delay: 1100 * time.Millisecond},
			},
			legacyWantTriggers: map[time.Duration]int64{
				450 * time.Millisecond: 0,
				// tick over at 500ms via maxWait
				550 * time.Millisecond: 1,
				950 * time.Millisecond: 1,
				// tick over at 1000ms via maxWait
				1050 * time.Millisecond: 2,
				1100 * time.Millisecond: 2,
				1150 * time.Millisecond: 2,
				1250 * time.Millisecond: 2,
				// tick over at 1300ms via wait (1100ms + 200ms)
				1350 * time.Millisecond: 3,
				// still 3 at at the end
				2000 * time.Millisecond: 3,
			},
		},
		{
			name: "two maxWaits, on cancel, and one wait expiry",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			legacyCalls: []testOp{
				{delay: 0 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 490 * time.Millisecond},
				// maxWait triggers
				{delay: 510 * time.Millisecond},
				{delay: 600 * time.Millisecond},
				{delay: 700 * time.Millisecond},
				{delay: 800 * time.Millisecond},
				{delay: 900 * time.Millisecond},
				{delay: 950 * time.Millisecond, reset: true},
				// wait and maxWait are both canceled
				{delay: 1510 * time.Millisecond},
				{delay: 1600 * time.Millisecond},
				{delay: 1700 * time.Millisecond},
				{delay: 1800 * time.Millisecond},
				{delay: 1900 * time.Millisecond},
				{delay: 1990 * time.Millisecond},
				// maxWait triggers
				{delay: 2010 * time.Millisecond},
				{delay: 2100 * time.Millisecond},
			},
			legacyWantTriggers: map[time.Duration]int64{
				450 * time.Millisecond: 0,
				// tick over at 500ms via maxWait
				550 * time.Millisecond:  1,
				1950 * time.Millisecond: 1,
				// tick over at 1000ms via maxWait
				2050 * time.Millisecond: 2,
				2100 * time.Millisecond: 2,
				2150 * time.Millisecond: 2,
				2250 * time.Millisecond: 2,
				// tick over at 1300ms (1100ms + 200ms)
				2350 * time.Millisecond: 3,
				// still 3 at at the end
				3000 * time.Millisecond: 3,
			},
		},
	}

	runTestCases(t, tests)
}
