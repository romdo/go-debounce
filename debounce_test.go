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
	name               string
	wait               time.Duration
	options            []Option
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
	call  bool
	reset bool
	want  int64
}

func runTestCases(t *testing.T, tests []testCase) {
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var startTime int64
			var invocations []int64
			mux := sync.Mutex{}

			f := func() {
				mux.Lock()
				defer mux.Unlock()

				offset := time.Now().UnixMilli() - startTime
				invocations = append(invocations, offset)
			}

			debouncedFunc, resetFunc := New(tt.wait, f, tt.options...)

			wg := sync.WaitGroup{}

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

			wg.Wait()

			// Wait a bit of extra time just to try and make sure there's
			// no lingering debounce left.
			time.Sleep(tt.wait * 2)

			assert.Len(t, invocations, len(tt.wantInvocations), "invocations")

			margin := tt.assertMargin
			if margin == 0 {
				margin = 30
			}

			for _, want := range tt.wantInvocations {
				found := -1
				for i, inv := range invocations {
					if want-margin < inv && want+margin > inv {
						found = i
						break
					}
				}

				assert.NotEqual(t, -1, found,
					"no invocation within %d ms of %d ms", margin, want,
				)

				if found != -1 {
					// Remove the invocation from the list.
					invocations = append(
						invocations[:found], invocations[found+1:]...,
					)
				}
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
			calls: []int64{
				100,
			},
			wantInvocations: []int64{
				300, // Trailing trigger from call at 100 milliseconds.
			},
		},
		{
			name: "two calls, two triggers",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 400,
			},
			wantInvocations: []int64{
				300, // Trailing trigger from call at 100 milliseconds.
				600, // Trailing trigger from call at 400 milliseconds.
			},
		},
		{
			name: "one burst of calls, one triggers",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			wantInvocations: []int64{
				700, // Trailing trigger from call at 500 milliseconds.
			},
		},
		{
			name: "one burst of calls with a reset, one trigger",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 150, 200, 250, 300, 350, 400,
				500, 550, 600,
			},
			resets: []int64{
				450,
			},
			wantInvocations: []int64{
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
		{
			name: "two close bursts of calls, two triggers",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			wantInvocations: []int64{
				400, // Trailing trigger from call at 200 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
		{
			name: "two close bursts of calls, longer wait, one trigger",
			wait: 400 * time.Millisecond,
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			wantInvocations: []int64{
				1000, // Trailing trigger from call at 600 milliseconds.
			},
		},
		{
			name: "two bursts of calls, two triggers",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 150, 200, 250, 300,
				800, 850, 900, 950, 1000,
			},
			wantInvocations: []int64{
				500,  // Trailing trigger from call at 300 milliseconds.
				1200, // Trailing trigger from call at 1000 milliseconds.
			},
		},
		{
			name: "two close bursts of calls, reset, two triggers",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			resets: []int64{
				450,
			},
			wantInvocations: []int64{
				400, // Trailing trigger from call at 200 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
	}
	runTestCases(t, tests)
}

func TestNew_with_Trailing(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:    "one call, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100,
			},
			wantInvocations: []int64{
				300, // Trailing trigger from call at 100 milliseconds
			},
		},
		{
			name:    "two calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 400,
			},
			wantInvocations: []int64{
				300, // Trailing trigger from call at 100 milliseconds.
				600, // Trailing trigger from call at 400 milliseconds.
			},
		},
		{
			name:    "one burst of calls, one triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			wantInvocations: []int64{
				700, // Trailing trigger from call at 500 milliseconds.
			},
		},
		{
			name:    "one burst of calls with a reset, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 150, 200, 250, 300, 350, 400,
				500, 550, 600,
			},
			resets: []int64{
				450,
			},
			wantInvocations: []int64{
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
		{
			name:    "two close bursts of calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			wantInvocations: []int64{
				400, // Trailing trigger from call at 200 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
		{
			name:    "two close bursts of calls, longer wait, one trigger",
			wait:    400 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			wantInvocations: []int64{
				1000, // Trailing trigger from call at 600 milliseconds.
			},
		},
		{
			name:    "two bursts of calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 150, 200, 250, 300,
				800, 850, 900, 950, 1000,
			},
			wantInvocations: []int64{
				500,  // Trailing trigger from call at 300 milliseconds.
				1200, // Trailing trigger from call at 1000 milliseconds.
			},
		},
		{
			name:    "two close bursts of calls, reset, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 150, 200,
				500, 550, 600,
			},
			resets: []int64{
				450,
			},
			wantInvocations: []int64{
				400, // Trailing trigger from call at 200 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
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
			calls: []int64{
				100,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100 milliseconds.
			},
		},
		{
			name:    "two calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 400,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100 milliseconds.
				400, // Leading trigger at 400 milliseconds.
			},
		},
		{
			name:    "one burst of calls, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 151, 200, 250, 300, 350, 400, 450, 500,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100 milliseconds.
			},
		},
		{
			name:    "one burst of calls with a reset, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 151, 200, 250, 300,
				400, 451, 500, 550,
			},
			resets: []int64{
				350,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100 milliseconds.
				400, // Leading trigger at 400 milliseconds.
			},
		},
		{
			name:    "two close bursts of calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 151, 200,
				500, 551, 600,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100 milliseconds.
				500, // Leading trigger at 500 milliseconds.
			},
		},
		{
			name:    "two close bursts of calls, longer wait, one trigger",
			wait:    500 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 151, 200,
				500, 551, 600,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100 milliseconds.
			},
		},
		{
			name:    "two bursts of calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 151, 200, 250, 300,
				600, 651, 700, 750, 800,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100 milliseconds.
				600, // Leading trigger at 600 milliseconds.
			},
		},
		{
			name:    "two close bursts of calls, reset, two triggers",
			wait:    500 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 151, 200,
				500, 551, 600,
			},
			resets: []int64{
				450,
			},
			wantInvocations: []int64{
				100, // Leading trigger at 100 milliseconds.
				500, // Leading trigger at 500 milliseconds.
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
				100, // Leading trigger at 100 milliseconds.
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
				100, // Leading trigger at 100 milliseconds.
				400, // Leading trigger at 400 milliseconds.
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
				100, // Leading trigger at 100 milliseconds.
				700, // Trailing trigger from call at 500 milliseconds.
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
				100, // Leading trigger at 100 milliseconds.
				500, // Leading trigger at 500 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
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
				100, // Leading trigger at 100 milliseconds.
				400, // Trailing trigger from call at 200 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
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
				100,  // Leading trigger at 100 milliseconds.
				1000, // Trailing trigger from call at 600 milliseconds.
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
				100,  // Leading trigger at 100 milliseconds.
				500,  // Trailing trigger from call at 300 milliseconds.
				800,  // Leading trigger at 800 milliseconds.
				1200, // Trailing trigger from call at 1000 milliseconds.
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
				100, // Leading trigger at 100 milliseconds.
				400, // Trailing trigger from call at 200 milliseconds.
				500, // Leading trigger at 500 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
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
			calls: []int64{
				0, 50, 70, 70, 150, 150,
			},
			wantInvocations: []int64{
				350, // Trailing trigger from call at 150 milliseconds.
			},
		},
		{
			name: "until right before maxWait",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 100, 200, 300, 400,
			},
			wantInvocations: []int64{
				500, // MaxWait trigger at 500 milliseconds.
			},
		},
		{
			name: "until right after maxWait",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 100, 200, 300, 400, 600,
			},
			wantInvocations: []int64{
				500, // MaxWait trigger at 500 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
		{
			name: "until two maxWaits and one wait expiry",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 100, 200, 300, 400, 490,
				510, 600, 700, 800, 900, 990,
				1010, 1100,
			},
			wantInvocations: []int64{
				500,  // MaxWait trigger at 500 milliseconds.
				1000, // MaxWait trigger at 1000 milliseconds.
				1300, // Trailing trigger from call at 1100 milliseconds.
			},
		},
		{
			name: "two maxWaits, on cancel, and one wait expiry",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 100, 200, 300, 400, 490,
				510, 600, 700, 800, 900,
				1510, 1600, 1700, 1800, 1900, 1990,
				2010, 2100,
			},
			resets: []int64{
				950,
			},
			wantInvocations: []int64{
				500,  // MaxWait trigger at 500 milliseconds.
				2010, // MaxWait trigger at 2010 milliseconds.
				2300, // Trailing trigger from call at 2100 milliseconds.
			},
		},
	}

	runTestCases(t, tests)
}
