package debounce

import (
	"flag"
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"
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

func getFuncName(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

type testCase struct {
	name        string
	mutable     bool
	wait        time.Duration
	options     []Option
	calls       []int64
	resets      []int64
	want        []int64
	wantMutable map[int64]int64
	margin      int64
}

type invocation struct {
	call int64
	time time.Time
	diff time.Duration
}

//nolint:gocyclo
func runTestCases(t *testing.T, tests []testCase) {
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var callCount int64 = -1
			invocations := []*invocation{}
			mux := sync.Mutex{}

			fn := func() {
				mux.Lock()
				defer mux.Unlock()

				invocations = append(invocations, &invocation{
					call: atomic.LoadInt64(&callCount),
					time: time.Now(),
				})
			}
			mutableFn := func(i int64) func() {
				return func() {
					mux.Lock()
					defer mux.Unlock()

					invocations = append(invocations, &invocation{
						call: i,
						time: time.Now(),
					})
				}
			}

			var mDeboucedFunc func(func())
			var debouncedFunc func()
			var resetFunc func()
			if tt.mutable {
				mDeboucedFunc, resetFunc = NewMutable(tt.wait, tt.options...)
			} else {
				debouncedFunc, resetFunc = New(tt.wait, fn, tt.options...)
			}
			wg := sync.WaitGroup{}
			startTime := time.Now()

			if tt.mutable {
				for i, offset := range tt.calls {
					i := i
					wg.Add(1)
					go func(i int64, x int64) {
						defer wg.Done()
						time.Sleep(time.Duration(x) * time.Millisecond)
						mDeboucedFunc(mutableFn(i))
					}(int64(i), offset)
				}
			} else {
				for _, offset := range tt.calls {
					wg.Add(1)
					go func(x int64) {
						defer wg.Done()
						time.Sleep(time.Duration(x) * time.Millisecond)
						atomic.AddInt64(&callCount, 1)
						debouncedFunc()
					}(offset)
				}
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

			margin := time.Duration(tt.margin) * time.Millisecond
			if margin == 0 {
				margin = 40 * time.Millisecond
			}

			for _, inv := range invocations {
				inv.diff = inv.time.Sub(startTime).Abs()
			}

			assert.Len(t, invocations, len(tt.want)+len(tt.wantMutable))

			if len(tt.want) > 0 {
				for _, want := range tt.want {
					// Find all invocations within the margin along with their
					// offset from the desired invocation time.
					wantTime := startTime.Add(
						time.Duration(want) * time.Millisecond,
					)

					found := make(map[int]time.Duration)
					for i, inv := range invocations {
						if wantTime.Before(inv.time.Add(margin)) &&
							wantTime.After(inv.time.Add(-margin)) {
							found[i] = wantTime.Sub(inv.time).Abs()
						}
					}

					assert.True(t, len(found) > 0,
						"no invocation within %s of %dms", margin, want,
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
			}
			if len(tt.wantMutable) > 0 {
				for i, want := range tt.wantMutable {
					var inv *invocation
					for _, v := range invocations {
						if v.call == i {
							inv = v
							break
						}
					}

					if !assert.NotNil(t, inv, "invocations[%d]", i) {
						continue
					}

					wantTime := startTime.Add(
						time.Duration(want) * time.Millisecond,
					)

					assert.WithinDuration(t,
						wantTime,
						inv.time,
						margin,
					)
				}
			}

			// NOTE: This is helpful when working on a failing test.
			// if t.Failed() {
			// 	spew.Dump(invocations)
			// }
		})
	}
}
