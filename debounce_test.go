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

type testOp struct {
	delay  time.Duration
	cancel bool
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		wait         time.Duration
		calls        []testOp
		wantTriggers map[time.Duration]int
	}{
		{
			name: "one call one trigger",
			wait: 20 * time.Millisecond,
			calls: []testOp{
				{delay: 10 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				5 * time.Millisecond:   0,
				15 * time.Millisecond:  0,
				25 * time.Millisecond:  0,
				35 * time.Millisecond:  1,
				150 * time.Millisecond: 1,
			},
		},
		{
			name: "two calls two triggers",
			wait: 20 * time.Millisecond,
			calls: []testOp{
				{delay: 10 * time.Millisecond},
				{delay: 40 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				5 * time.Millisecond:  0,
				15 * time.Millisecond: 0,
				25 * time.Millisecond: 0,
				// from first call at 10ms (+20ms wait = 30ms)
				35 * time.Millisecond: 1,
				55 * time.Millisecond: 1,
				// from second call at 40ms (+20ms wait = 60ms)
				65 * time.Millisecond:  2,
				150 * time.Millisecond: 2,
			},
		},
		{
			name: "many calls two triggers",
			wait: 20 * time.Millisecond,
			calls: []testOp{
				{delay: 5 * time.Millisecond},
				{delay: 5 * time.Millisecond},
				{delay: 10 * time.Millisecond}, // trigger 1
				{delay: 35 * time.Millisecond},
				{delay: 40 * time.Millisecond},
				{delay: 45 * time.Millisecond},
				{delay: 45 * time.Millisecond},
				{delay: 50 * time.Millisecond}, // trigger 2
			},
			wantTriggers: map[time.Duration]int{
				5 * time.Millisecond:  0,
				15 * time.Millisecond: 0,
				25 * time.Millisecond: 0,
				// from call at 10ms (+20ms wait = 30ms)
				35 * time.Millisecond: 1,
				65 * time.Millisecond: 1,
				// from call at 50ms (+20ms wait = 70ms)
				75 * time.Millisecond:  2,
				150 * time.Millisecond: 2,
			},
		},
		{
			name: "many calls, one cancel, two triggers",
			wait: 20 * time.Millisecond,
			calls: []testOp{
				{delay: 5 * time.Millisecond},
				{delay: 5 * time.Millisecond},
				{delay: 10 * time.Millisecond}, // trigger 1
				{delay: 35 * time.Millisecond},
				{delay: 40 * time.Millisecond},
				{delay: 50 * time.Millisecond, cancel: true},
				{delay: 80 * time.Millisecond},
				{delay: 90 * time.Millisecond},
				{delay: 100 * time.Millisecond}, // trigger 2
			},
			wantTriggers: map[time.Duration]int{
				5 * time.Millisecond:  0,
				15 * time.Millisecond: 0,
				25 * time.Millisecond: 0,
				// from call at 10ms (+20ms wait = 30ms)
				35 * time.Millisecond:  1,
				115 * time.Millisecond: 1,
				// call at 100ms (+20ms wait = 120ms)
				125 * time.Millisecond: 2,
				150 * time.Millisecond: 2,
			},
		},
		{
			name: "many triggers within wait time",
			wait: 20 * time.Millisecond,
			calls: []testOp{
				{delay: 10 * time.Millisecond},
				{delay: 20 * time.Millisecond},
				{delay: 30 * time.Millisecond},
				{delay: 40 * time.Millisecond},
				{delay: 50 * time.Millisecond},
				{delay: 60 * time.Millisecond},
				{delay: 70 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				85 * time.Millisecond:  0,
				95 * time.Millisecond:  1,
				150 * time.Millisecond: 1,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := sync.RWMutex{}

			n := 0
			d, c := New(tt.wait, func() {
				mux.Lock()
				defer mux.Unlock()
				n++
			})

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

func TestNewWithMaxWait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		wait         time.Duration
		maxwait      time.Duration
		calls        []testOp
		wantTriggers map[time.Duration]int
	}{
		{
			name:    "all within wait time",
			wait:    20 * time.Millisecond,
			maxwait: 50 * time.Millisecond,
			calls: []testOp{
				{delay: 0 * time.Millisecond},
				{delay: 5 * time.Millisecond},
				{delay: 7 * time.Millisecond},
				{delay: 7 * time.Millisecond},
				{delay: 15 * time.Millisecond},
				{delay: 15 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				30 * time.Millisecond: 0,
				// tick over at 35ms (15ms + 20ms)
				40 * time.Millisecond: 1,
				// still 1 at at the end
				100 * time.Millisecond: 1,
			},
		},
		{
			name:    "until right before maxWait",
			wait:    20 * time.Millisecond,
			maxwait: 50 * time.Millisecond,
			calls: []testOp{
				{delay: 0 * time.Millisecond},
				{delay: 10 * time.Millisecond},
				{delay: 20 * time.Millisecond},
				{delay: 30 * time.Millisecond},
				{delay: 40 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				45 * time.Millisecond: 0,
				// tick over at 50ms via maxWait
				55 * time.Millisecond: 1,
				// still 1 at at the end
				100 * time.Millisecond: 1,
			},
		},
		{
			name:    "until right after maxWait",
			wait:    20 * time.Millisecond,
			maxwait: 50 * time.Millisecond,
			calls: []testOp{
				{delay: 0 * time.Millisecond},
				{delay: 10 * time.Millisecond},
				{delay: 20 * time.Millisecond},
				{delay: 30 * time.Millisecond},
				{delay: 40 * time.Millisecond},
				{delay: 60 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				45 * time.Millisecond: 0,
				// tick over at 50ms via maxWait
				55 * time.Millisecond: 1,
				75 * time.Millisecond: 1,
				// tick over at 80ms (60ms + 20ms)
				85 * time.Millisecond: 2,
				// still 2 at at the end
				150 * time.Millisecond: 2,
			},
		},
		{
			name:    "until two maxWaits and one wait exipry",
			wait:    20 * time.Millisecond,
			maxwait: 50 * time.Millisecond,
			calls: []testOp{
				{delay: 0 * time.Millisecond},
				{delay: 10 * time.Millisecond},
				{delay: 20 * time.Millisecond},
				{delay: 30 * time.Millisecond},
				{delay: 40 * time.Millisecond},
				{delay: 49 * time.Millisecond},
				// maxWait triggers at 50ms (0ms + 50ms)
				{delay: 51 * time.Millisecond},
				{delay: 60 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 80 * time.Millisecond},
				{delay: 90 * time.Millisecond},
				{delay: 99 * time.Millisecond},
				// maxWait triggers at 100ms (50ms + 50ms)
				{delay: 101 * time.Millisecond},
				{delay: 110 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				45 * time.Millisecond: 0,
				// tick over at 50ms via maxWait
				55 * time.Millisecond: 1,
				95 * time.Millisecond: 1,
				// tick over at 100ms via maxWait
				105 * time.Millisecond: 2,
				110 * time.Millisecond: 2,
				115 * time.Millisecond: 2,
				125 * time.Millisecond: 2,
				// tick over at 130ms via wait (110ms + 20ms)
				135 * time.Millisecond: 3,
				// still 3 at at the end
				200 * time.Millisecond: 3,
			},
		},
		{
			name:    "two maxWaits, on cancel, and one wait expiry",
			wait:    20 * time.Millisecond,
			maxwait: 50 * time.Millisecond,
			calls: []testOp{
				{delay: 0 * time.Millisecond},
				{delay: 10 * time.Millisecond},
				{delay: 20 * time.Millisecond},
				{delay: 30 * time.Millisecond},
				{delay: 40 * time.Millisecond},
				{delay: 49 * time.Millisecond},
				// maxWait triggers
				{delay: 51 * time.Millisecond},
				{delay: 60 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 80 * time.Millisecond},
				{delay: 90 * time.Millisecond},
				{delay: 95 * time.Millisecond, cancel: true},
				// wait and maxWait are both canceled
				{delay: 151 * time.Millisecond},
				{delay: 160 * time.Millisecond},
				{delay: 170 * time.Millisecond},
				{delay: 180 * time.Millisecond},
				{delay: 190 * time.Millisecond},
				{delay: 199 * time.Millisecond},
				// maxWait triggers
				{delay: 201 * time.Millisecond},
				{delay: 210 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				45 * time.Millisecond: 0,
				// tick over at 50ms via maxWait
				55 * time.Millisecond:  1,
				195 * time.Millisecond: 1,
				// tick over at 100ms via maxWait
				205 * time.Millisecond: 2,
				210 * time.Millisecond: 2,
				215 * time.Millisecond: 2,
				225 * time.Millisecond: 2,
				// tick over at 130ms (110ms + 20ms)
				235 * time.Millisecond: 3,
				// still 3 at at the end
				300 * time.Millisecond: 3,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := sync.RWMutex{}

			n := 0
			d, c := NewWithMaxWait(tt.wait, tt.maxwait, func() {
				mux.Lock()
				defer mux.Unlock()
				n++
			})

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
