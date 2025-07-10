package debounce

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMutable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		wait         time.Duration
		calls        []testOp
		wantTriggers map[time.Duration]int
		wantFuncs    []int
	}{
		{
			name: "one trigger",
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
			wantFuncs: []int{0},
		},
		{
			name: "two separate triggers",
			wait: 200 * time.Millisecond,
			calls: []testOp{
				{delay: 100 * time.Millisecond},
				{delay: 400 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond:  0,
				150 * time.Millisecond: 0,
				250 * time.Millisecond: 0,
				// from call at 100ms (+200ms wait = 300ms)
				350 * time.Millisecond: 1,
				550 * time.Millisecond: 1,
				// from call at 400ms (+200ms wait = 600ms)
				650 * time.Millisecond: 2,
				// still 2 at at the end
				1150 * time.Millisecond: 2,
			},
			wantFuncs: []int{0, 1},
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
				{delay: 500 * time.Millisecond, reset: true},
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
				// from call at 1000ms (+200ms wait = 1200ms)
				1250 * time.Millisecond: 2,
				// still 2 at at the end
				1750 * time.Millisecond: 2,
			},
			wantFuncs: []int{2, 8},
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
				850 * time.Millisecond: 0,
				950 * time.Millisecond: 1,
				// still 1 at at the end
				1450 * time.Millisecond: 1,
			},
			wantFuncs: []int{6},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := sync.RWMutex{}

			n := 0
			got := []int{}

			d, c := NewMutable(tt.wait)

			wg := sync.WaitGroup{}
			for i, op := range tt.calls {
				i := i
				wg.Add(1)
				go func(delay time.Duration, cancel bool) {
					defer wg.Done()
					time.Sleep(delay)

					if cancel {
						c()
					} else {
						d(func() {
							mux.Lock()
							defer mux.Unlock()
							n++
							got = append(got, i)
						})
					}
				}(op.delay, op.reset)
			}

			for interval, count := range tt.wantTriggers {
				wg.Add(1)
				go func(interval time.Duration, count int) {
					defer wg.Done()
					time.Sleep(interval)

					mux.RLock()
					defer mux.RUnlock()
					assert.Equal(t, count, n, "at %s", interval)
				}(interval, count)
			}

			wg.Wait()

			assert.Equal(t, tt.wantFuncs, got)
		})
	}
}

func TestNewMutableAndMaxWait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		wait         time.Duration
		maxwait      time.Duration
		calls        []testOp
		wantTriggers map[time.Duration]int
		wantFuncs    []int
	}{
		{
			name:    "all within wait time",
			wait:    200 * time.Millisecond,
			maxwait: 500 * time.Millisecond,
			calls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 20 * time.Millisecond},
				{delay: 40 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 150 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				300 * time.Millisecond: 0,
				// tick over at 350ms (150ms + 200ms)
				400 * time.Millisecond: 1,
				// still 1 at at the end
				900 * time.Millisecond: 1,
			},
			wantFuncs: []int{5},
		},
		{
			name:    "until right before maxWait",
			wait:    200 * time.Millisecond,
			maxwait: 500 * time.Millisecond,
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
				1050 * time.Millisecond: 1,
			},
			wantFuncs: []int{4},
		},
		{
			name:    "until right after maxWait",
			wait:    200 * time.Millisecond,
			maxwait: 500 * time.Millisecond,
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
				1350 * time.Millisecond: 2,
			},
			wantFuncs: []int{4, 5},
		},
		{
			name:    "until two maxWaits and one wait expiry",
			wait:    200 * time.Millisecond,
			maxwait: 500 * time.Millisecond,
			calls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				// maxWait triggers at 500ms (00ms + 500ms)
				{delay: 520 * time.Millisecond},
				{delay: 600 * time.Millisecond},
				{delay: 700 * time.Millisecond},
				{delay: 800 * time.Millisecond},
				{delay: 900 * time.Millisecond},
				// maxWait triggers at 1020ms (520ms + 500ms)
				{delay: 1050 * time.Millisecond},
				{delay: 1100 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				450 * time.Millisecond: 0,
				// tick over at 500ms via maxWait
				550 * time.Millisecond: 1,
				950 * time.Millisecond: 1,
				// tick over at 1020ms via maxWait
				1050 * time.Millisecond: 2,
				1100 * time.Millisecond: 2,
				1150 * time.Millisecond: 2,
				1250 * time.Millisecond: 2,
				// tick over at 1300ms wait (1100ms + 200ms)
				1350 * time.Millisecond: 3,
				// still 3 at at the end
				1850 * time.Millisecond: 3,
			},
			wantFuncs: []int{4, 9, 11},
		},
		{
			name:    "two maxWaits, on cancel, and one wait expiry",
			wait:    200 * time.Millisecond,
			maxwait: 500 * time.Millisecond,
			calls: []testOp{
				{delay: 0o0 * time.Millisecond},
				{delay: 100 * time.Millisecond},
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				// maxWait triggers
				{delay: 520 * time.Millisecond},
				{delay: 600 * time.Millisecond},
				{delay: 700 * time.Millisecond},
				{delay: 800 * time.Millisecond},
				{delay: 900 * time.Millisecond},
				{delay: 950 * time.Millisecond, reset: true},
				// wait and maxWait are both canceled
				{delay: 1530 * time.Millisecond},
				{delay: 1600 * time.Millisecond},
				{delay: 1700 * time.Millisecond},
				{delay: 1800 * time.Millisecond},
				{delay: 1900 * time.Millisecond},
				// maxWait triggers
				{delay: 2030 * time.Millisecond},
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
				2850 * time.Millisecond: 3,
			},
			wantFuncs: []int{4, 16, 17},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := sync.RWMutex{}

			n := 0
			got := []int{}

			d, c := NewMutableWithMaxWait(tt.wait, tt.maxwait)

			wg := sync.WaitGroup{}
			for i, op := range tt.calls {
				i := i
				wg.Add(1)
				go func(interval time.Duration, cancel bool) {
					defer wg.Done()
					time.Sleep(interval)

					if cancel {
						c()
					} else {
						d(func() {
							mux.Lock()
							defer mux.Unlock()
							n++
							got = append(got, i)
						})
					}
				}(op.delay, op.reset)
			}

			for interval, count := range tt.wantTriggers {
				wg.Add(1)
				go func(interval time.Duration, count int) {
					defer wg.Done()
					time.Sleep(interval)

					mux.RLock()
					defer mux.RUnlock()
					assert.Equal(t, count, n, "at %s", interval)
				}(interval, count)
			}

			wg.Wait()

			assert.Equal(t, tt.wantFuncs, got)
		})
	}
}
