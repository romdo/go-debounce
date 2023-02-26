package debounce

import (
	"flag"
	"fmt"
	"os"
	"sync"
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
	name         string
	wait         time.Duration
	options      []Option
	calls        []testOp
	wantTriggers map[time.Duration]int
}

type testOp struct {
	delay  time.Duration
	cancel bool
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name: "one call one trigger",
			wait: 200 * time.Millisecond,
			calls: []testOp{
				{delay: 100 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond:  0,
				150 * time.Millisecond: 0,
				250 * time.Millisecond: 0,
				350 * time.Millisecond: 1,
				// still 1 at at the end
				850 * time.Millisecond: 1,
			},
		},
		{
			name: "two calls two triggers",
			wait: 200 * time.Millisecond,
			calls: []testOp{
				{delay: 100 * time.Millisecond},
				{delay: 400 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond:  0,
				150 * time.Millisecond: 0,
				250 * time.Millisecond: 0,
				// from first call at 100ms (+200ms wait = 300ms)
				350 * time.Millisecond: 1,
				550 * time.Millisecond: 1,
				// from second call at 400ms (+200ms wait = 600ms)
				650 * time.Millisecond: 2,
				// still 2 at at the end
				1150 * time.Millisecond: 2,
			},
		},
		{
			name: "many calls two triggers",
			wait: 200 * time.Millisecond,
			calls: []testOp{
				{delay: 50 * time.Millisecond},
				{delay: 50 * time.Millisecond},
				{delay: 100 * time.Millisecond}, // trigger 1
				{delay: 350 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 450 * time.Millisecond},
				{delay: 450 * time.Millisecond},
				{delay: 500 * time.Millisecond}, // trigger 2
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond:  0,
				150 * time.Millisecond: 0,
				250 * time.Millisecond: 0,
				// from call at 100ms (+200ms wait = 300ms)
				350 * time.Millisecond: 1,
				650 * time.Millisecond: 1,
				// from call at 500ms (+200ms wait = 700ms)
				750 * time.Millisecond: 2,
				// still 2 at at the end
				1250 * time.Millisecond: 2,
			},
		},
		{
			name: "many calls, one cancel, one trigger",
			wait: 200 * time.Millisecond,
			calls: []testOp{
				{delay: 50 * time.Millisecond},
				{delay: 50 * time.Millisecond},
				{delay: 100 * time.Millisecond}, // trigger 1
				{delay: 350 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 500 * time.Millisecond, cancel: true}, // trigger 2
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond:  0,
				150 * time.Millisecond: 0,
				250 * time.Millisecond: 0,
				// from call at 100ms (+200ms wait = 300ms)
				350 * time.Millisecond: 1,
				450 * time.Millisecond: 1,
				// canceled at 500ms
				550 * time.Millisecond: 1,
				// still 1 at at the end
				1050 * time.Millisecond: 1,
			},
		},
		{
			name: "many calls, one cancel, two triggers",
			wait: 200 * time.Millisecond,
			calls: []testOp{
				{delay: 50 * time.Millisecond},
				{delay: 50 * time.Millisecond},
				{delay: 100 * time.Millisecond}, // trigger 1
				{delay: 350 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 500 * time.Millisecond, cancel: true},
				{delay: 800 * time.Millisecond},
				{delay: 900 * time.Millisecond},
				{delay: 1000 * time.Millisecond}, // trigger 2
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond:  0,
				150 * time.Millisecond: 0,
				250 * time.Millisecond: 0,
				// from call at 100ms (+200ms wait = 300ms)
				350 * time.Millisecond:  1,
				1150 * time.Millisecond: 1,
				// call at 1000ms (+200ms wait = 1200ms)
				1250 * time.Millisecond: 2,
				// still 1 at at the end
				1750 * time.Millisecond: 2,
			},
		},
		{
			name: "many triggers within wait time",
			wait: 200 * time.Millisecond,
			calls: []testOp{
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 500 * time.Millisecond},
				{delay: 600 * time.Millisecond},
				{delay: 700 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				850 * time.Millisecond:  0,
				950 * time.Millisecond:  1,
				1450 * time.Millisecond: 1,
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
			calls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 50 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 150 * time.Millisecond},
				{delay: 150 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
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
			calls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
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
			calls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 600 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
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
			calls: []testOp{
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
			wantTriggers: map[time.Duration]int{
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
			calls: []testOp{
				{delay: 0o0 * time.Millisecond},
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
				{delay: 950 * time.Millisecond, cancel: true},
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
			wantTriggers: map[time.Duration]int{
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

func runTestCases(t *testing.T, tests []testCase) {
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := sync.RWMutex{}

			n := 0
			f := func() {
				mux.Lock()
				defer mux.Unlock()
				n++
			}
			d, c := New(tt.wait, f, tt.options...)

			wg := sync.WaitGroup{}
			for _, op := range tt.calls {
				wg.Add(1)
				go func(delay time.Duration, cancel bool) {
					defer wg.Done()
					time.Sleep(delay)
					if cancel {
						c()
					} else {
						d()
					}
				}(op.delay, op.cancel)
			}

			for delay, count := range tt.wantTriggers {
				wg.Add(1)
				go func(interval time.Duration, count int) {
					defer wg.Done()
					time.Sleep(interval)

					mux.RLock()
					defer mux.RUnlock()
					assert.Equal(t, count, n, "at %s", interval)
				}(delay, count)
			}

			wg.Wait()
		})
	}
}
