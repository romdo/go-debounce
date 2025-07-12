package debounce

import (
	"flag"
	"fmt"
	"math"
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
	name    string
	wait    time.Duration
	options []Option
	calls   []int64
	resets  []int64
	want    []int64
	margin  int64
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

			// Get the longest between wait and maxWait, and multiply by 3 to
			// make sure there's no lingering debounce left.
			d := &Debouncer{wait: tt.wait}
			for _, opt := range tt.options {
				opt(d)
			}
			maxDelay := time.Duration(
				math.Max(float64(d.wait), float64(d.maxWait)),
			)
			// For tests with small wait durations, we want to make sure there's
			// enough time for the debounce to trigger.
			if maxDelay < 100*time.Millisecond {
				maxDelay = 100 * time.Millisecond
			}
			time.Sleep(maxDelay * 3)

			mux.Lock()
			defer mux.Unlock()

			assert.Len(t, invocations, len(tt.want), "invocations")

			margin := tt.margin
			if margin == 0 {
				margin = 40
			}

			for _, want := range tt.want {
				// Find all invocations within the margin along with their
				// offset from the desired invocation time.
				found := make(map[int]int64)
				for i, inv := range invocations {
					if want-margin < inv && want+margin > inv {
						found[i] = int64(math.Abs(float64(want - inv)))
					}
				}

				assert.True(t, len(found) > 0,
					"no invocation within %d ms of %d ms", margin, want,
				)

				if len(found) > 0 {
					// Determine the closest invocation.
					closestIndex := -1
					closestOffset := found[0]
					for i, offset := range found {
						if offset < closestOffset {
							closestIndex = i
							closestOffset = offset
						}
					}

					// Remove the closest invocation.
					if closestIndex != -1 {
						invocations = append(
							invocations[:closestIndex],
							invocations[closestIndex+1:]...,
						)
					}
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returned functions are Debounce and Reset from *Debouncer",
		func(t *testing.T) {
			t.Parallel()
			d := &Debouncer{}
			debouncedFunc, resetFunc := New(d.wait, func() {})

			assert.Equal(t, getFuncName(d.Debounce), getFuncName(debouncedFunc))
			assert.Equal(t, getFuncName(d.Reset), getFuncName(resetFunc))
		},
	)

	tests := []testCase{
		{
			name: "zero wait duration, immediate trigger",
			wait: 0,
			calls: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
		},
		{
			name: "negative wait duration, immediate trigger",
			wait: -100 * time.Millisecond,
			calls: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
		},
		{
			name: "one call, one trigger",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100,
			},
			want: []int64{
				300, // Trailing trigger from call at 100 milliseconds.
			},
		},
		{
			name: "two calls, two triggers",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 400,
			},
			want: []int64{
				300, // Trailing trigger from call at 100 milliseconds.
				600, // Trailing trigger from call at 400 milliseconds.
			},
		},
		{
			name: "one burst of calls, one trigger",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				700, // Trailing trigger from call at 500 milliseconds.
			},
		},
		{
			name: "one burst of parallel calls, one trigger",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
			},
			want: []int64{
				300, // Trailing trigger from calls at 100 milliseconds.
			},
		},
		{
			name: "one burst of calls ending with parallel calls, one trigger",
			wait: 200 * time.Millisecond,
			calls: []int64{
				100, 150, 200, 250, 300, 300, 300, 300, 300, 300, 300,
			},
			want: []int64{
				500, // Trailing trigger from calls at 300 milliseconds.
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
				400, // Trailing trigger from call at 200 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
	}
	runTestCases(t, tests)
}

func TestNew_withTrailing(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:    "zero wait duration, immediate trigger",
			wait:    0,
			options: []Option{Trailing()},
			calls: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
		},
		{
			name:    "negative wait duration, immediate trigger",
			wait:    -100 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
		},
		{
			name:    "one call, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100,
			},
			want: []int64{
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
			want: []int64{
				300, // Trailing trigger from call at 100 milliseconds.
				600, // Trailing trigger from call at 400 milliseconds.
			},
		},
		{
			name:    "one burst of calls, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				700, // Trailing trigger from call at 500 milliseconds.
			},
		},
		{
			name:    "one burst of parallel calls, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
			},
			want: []int64{
				300, // Trailing trigger from calls at 100 milliseconds.
			},
		},
		{
			name: "one burst of calls ending with parallel calls, " +
				"one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Trailing()},
			calls: []int64{
				100, 150, 200, 250, 300, 300, 300, 300, 300, 300, 300,
			},
			want: []int64{
				500, // Trailing trigger from calls at 300 milliseconds.
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
				400, // Trailing trigger from call at 200 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
	}
	runTestCases(t, tests)
}

func TestNew_withLeading(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:    "zero wait duration, immediate trigger",
			wait:    0,
			options: []Option{Leading()},
			calls: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
		},
		{
			name:    "negative wait duration, immediate trigger",
			wait:    -100 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
		},
		{
			name:    "one call, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100,
			},
			want: []int64{
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
			want: []int64{
				100, // Leading trigger at 100 milliseconds.
				400, // Leading trigger at 400 milliseconds.
			},
		},
		{
			name:    "one burst of calls, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				100, // Leading trigger at 100 milliseconds.
			},
		},
		{
			name:    "one burst of parallel calls, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
			},
			want: []int64{
				100, // Leading trigger at 100 milliseconds.
			},
		},
		{
			name: "one burst of calls ending with parallel calls, " +
				"one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading()},
			calls: []int64{
				100, 150, 200, 250, 300, 300, 300, 300, 300, 300, 300,
			},
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
				100, // Leading trigger at 100 milliseconds.
				500, // Leading trigger at 500 milliseconds.
			},
		},
	}
	runTestCases(t, tests)
}

func TestNew_withLeadingAndTrailing(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:    "zero wait duration, immediate trigger",
			wait:    0,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
		},
		{
			name:    "negative wait duration, immediate trigger",
			wait:    -100 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
			want: []int64{
				0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
			},
		},
		{
			name:    "one call, one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100,
			},
			want: []int64{
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
			want: []int64{
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
			want: []int64{
				100, // Leading trigger at 100 milliseconds.
				700, // Trailing trigger from call at 500 milliseconds.
			},
		},
		{
			name:    "one burst of parallel calls, two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
			},
			want: []int64{
				100, // Leading trigger at 100 milliseconds.
				300, // Trailing trigger from calls at 100 milliseconds.
			},
		},
		{
			name: "one burst of calls ending with parallel calls, " +
				"two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []int64{
				100, 150, 200, 250, 300, 300, 300, 300, 300, 300, 300,
			},
			want: []int64{
				100, // Leading trigger at 100 milliseconds.
				500, // Trailing trigger from calls at 300 milliseconds.
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
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
			want: []int64{
				100, // Leading trigger at 100 milliseconds.
				400, // Trailing trigger from call at 200 milliseconds.
				500, // Leading trigger at 500 milliseconds.
				800, // Trailing trigger from call at 600 milliseconds.
			},
		},
	}

	runTestCases(t, tests)
}

func TestNew_withMaxWait(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name: "zero maxWait duration, maxWait is ignored",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(0),
			},
			calls: []int64{
				0, 50, 150, 250, 350, 450,
			},
			want: []int64{
				650, // Trailing trigger from call at 450 milliseconds.
			},
		},
		{
			name: "negative maxWait duration, maxWait is ignored",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(-100 * time.Millisecond),
			},
			calls: []int64{
				0, 50, 150, 250, 350, 450,
			},
			want: []int64{
				650, // Trailing trigger from call at 450 milliseconds.
			},
		},
		{
			name: "maxWait shorter than wait duration, maxWait is ignored",
			wait: 500 * time.Millisecond,
			options: []Option{
				MaxWait(200 * time.Millisecond),
			},
			calls: []int64{
				0, 100, 200, 300, 400,
			},
			want: []int64{
				900, // Trailing trigger from call at 400 milliseconds.
			},
		},
		{
			name: "maxWait equal to wait duration, maxWait is ignored",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(200 * time.Millisecond),
			},
			calls: []int64{
				0, 100, 200, 300, 400,
			},
			want: []int64{
				600, // Trailing trigger from call at 400 milliseconds.
			},
		},
		{
			name: "maxWait is slightly longer than wait duration",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(201 * time.Millisecond),
			},
			calls: []int64{
				0, 100, 200, 300, 400,
			},
			want: []int64{
				201, // Max wait trigger via call at 0 milliseconds.
				600, // Trailing trigger from call at 400 milliseconds.
			},
		},
		{
			name: "one burst within wait time",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 50, 70, 70, 150, 150,
			},
			want: []int64{
				350, // Trailing trigger from call at 150 milliseconds.
			},
		},
		{
			name: "one burst until right before maxWait",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 50, 150, 250, 350, 450,
			},
			want: []int64{
				500, // Max wait trigger via call at 0 milliseconds.
			},
		},
		{
			name: "one burst until right after maxWait",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 50, 150, 250, 350, 450, 550,
			},
			want: []int64{
				500, // Max wait trigger via call at 0 milliseconds.
				750, // Trailing trigger from call at 550 milliseconds.
			},
		},
		{
			name: "one burst across two maxWaits and one trailing trigger",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 50, 150, 250, 350, 450, 550, 650, 750, 850, 950, 1050, 1150,
			},
			want: []int64{
				500,  // Max wait trigger via call at 0 milliseconds.
				1050, // Max wait trigger via call at 550 milliseconds.
				1350, // Trailing trigger from call at 1150 milliseconds.
			},
		},
		{
			name: "two bursts with a maxWait between them",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 100, 200, 300, 400,
				600, 700, 800,
			},
			want: []int64{
				500,  // Max wait trigger via call at 0 milliseconds.
				1000, // Trailing trigger from call at 800 milliseconds.
			},
		},
		{
			name: "two bursts with maxWaits, reset, and trailing trigger",
			wait: 200 * time.Millisecond,
			options: []Option{
				MaxWait(500 * time.Millisecond),
			},
			calls: []int64{
				0, 50, 150, 250, 350, 450, 550, 650, 750, 850,
				1550, 1650, 1750, 1850, 1950, 2050, 2150,
			},
			resets: []int64{
				950,
			},
			want: []int64{
				500,  // Max wait trigger via call at 0 milliseconds.
				2050, // Max wait trigger via call at 1550 milliseconds.
				2350, // Trailing trigger from call at 2150 milliseconds.
			},
		},
	}

	runTestCases(t, tests)
}

func TestNewMutable(t *testing.T) {
	t.Parallel()

	t.Run("returned functions are DebounceWith and Reset from *Debouncer",
		func(t *testing.T) {
			t.Parallel()
			d := &Debouncer{}
			debouncedFunc, resetFunc := NewMutable(100 * time.Millisecond)

			assert.Equal(t,
				getFuncName(d.DebounceWith),
				getFuncName(debouncedFunc),
			)
			assert.Equal(t, getFuncName(d.Reset), getFuncName(resetFunc))
		},
	)

	t.Run("last function wins", func(t *testing.T) {
		t.Parallel()

		var target int32

		debouncedFunc, _ := NewMutable(100 * time.Millisecond)

		debouncedFunc(func() { atomic.AddInt32(&target, 1) })
		debouncedFunc(func() { atomic.AddInt32(&target, 2) })
		debouncedFunc(func() { atomic.AddInt32(&target, 4) })

		time.Sleep(200 * time.Millisecond)

		assert.Equal(t, int32(4), atomic.LoadInt32(&target))
	})
}
