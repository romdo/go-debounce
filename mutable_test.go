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
			wantFuncs: []int{0},
		},
		{
			name: "two separate triggers",
			wait: 20 * time.Millisecond,
			calls: []testOp{
				{delay: 10 * time.Millisecond},
				{delay: 40 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				5 * time.Millisecond:  0,
				15 * time.Millisecond: 0,
				25 * time.Millisecond: 0,
				// from call at 10ms (+20ms wait = 30ms)
				35 * time.Millisecond: 1,
				55 * time.Millisecond: 1,
				// from call at 40ms (+20ms wait = 60ms)
				65 * time.Millisecond:  2,
				150 * time.Millisecond: 2,
			},
			wantFuncs: []int{0, 1},
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
				// from call at 100ms (+20ms wait = 120ms)
				125 * time.Millisecond: 2,
				150 * time.Millisecond: 2,
			},
			wantFuncs: []int{2, 8},
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
				}(op.delay, op.cancel)
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
			wait:    20 * time.Millisecond,
			maxwait: 50 * time.Millisecond,
			calls: []testOp{
				{delay: 0 * time.Millisecond},
				{delay: 2 * time.Millisecond},
				{delay: 4 * time.Millisecond},
				{delay: 7 * time.Millisecond},
				{delay: 10 * time.Millisecond},
				{delay: 15 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				30 * time.Millisecond: 0,
				// tick over at 35ms (15ms + 20ms)
				40 * time.Millisecond: 1,
				// still 1 at at the end
				100 * time.Millisecond: 1,
			},
			wantFuncs: []int{5},
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
			wantFuncs: []int{4},
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
			wantFuncs: []int{4, 5},
		},
		{
			name:    "until two maxWaits and one wait expiry",
			wait:    20 * time.Millisecond,
			maxwait: 50 * time.Millisecond,
			calls: []testOp{
				{delay: 0 * time.Millisecond},
				{delay: 10 * time.Millisecond},
				{delay: 20 * time.Millisecond},
				{delay: 30 * time.Millisecond},
				{delay: 40 * time.Millisecond},
				// maxWait triggers at 50ms (0ms + 50ms)
				{delay: 52 * time.Millisecond},
				{delay: 60 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 80 * time.Millisecond},
				{delay: 90 * time.Millisecond},
				// maxWait triggers at 102ms (52ms + 50ms)
				{delay: 105 * time.Millisecond},
				{delay: 110 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				45 * time.Millisecond: 0,
				// tick over at 50ms via maxWait
				55 * time.Millisecond: 1,
				95 * time.Millisecond: 1,
				// tick over at 102ms via maxWait
				105 * time.Millisecond: 2,
				110 * time.Millisecond: 2,
				115 * time.Millisecond: 2,
				125 * time.Millisecond: 2,
				// tick over at 130ms wait (110ms + 20ms)
				135 * time.Millisecond: 3,
				// still 3 at at the end
				200 * time.Millisecond: 3,
			},
			wantFuncs: []int{4, 9, 11},
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
				// maxWait triggers
				{delay: 52 * time.Millisecond},
				{delay: 60 * time.Millisecond},
				{delay: 70 * time.Millisecond},
				{delay: 80 * time.Millisecond},
				{delay: 90 * time.Millisecond},
				{delay: 95 * time.Millisecond, cancel: true},
				// wait and maxWait are both canceled
				{delay: 153 * time.Millisecond},
				{delay: 160 * time.Millisecond},
				{delay: 170 * time.Millisecond},
				{delay: 180 * time.Millisecond},
				{delay: 190 * time.Millisecond},
				// maxWait triggers
				{delay: 203 * time.Millisecond},
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
				}(op.delay, op.cancel)
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
