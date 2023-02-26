package debounce

import (
	"testing"
	"time"
)

func TestNew_with_Leading_and_Trailing(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:    "one call one trigger",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []testOp{
				{delay: 100 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond:  0,
				150 * time.Millisecond: 1,
				// still 1 at at the end
				650 * time.Millisecond: 1,
			},
		},
		{
			name:    "two calls two triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []testOp{
				{delay: 100 * time.Millisecond},
				{delay: 400 * time.Millisecond},
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond: 0,
				// call at 100ms
				150 * time.Millisecond: 1,
				250 * time.Millisecond: 1,
				350 * time.Millisecond: 1,
				// call at 400ms
				450 * time.Millisecond: 2,
				// still 2 at at the end
				950 * time.Millisecond: 2,
			},
		},
		{
			name:    "many calls three triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []testOp{
				{delay: 100 * time.Millisecond}, // trigger 1
				{delay: 150 * time.Millisecond},
				{delay: 200 * time.Millisecond}, // trigger 2 delayed to 400ms
				{delay: 500 * time.Millisecond}, // skipped, too close to 400ms
				{delay: 550 * time.Millisecond},
				{delay: 560 * time.Millisecond},
				{delay: 600 * time.Millisecond}, // trigger 2 at 80ms
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond: 0,
				// call at 100ms
				150 * time.Millisecond: 1,
				350 * time.Millisecond: 1,
				// from call at 200ms (+200ms wait = 400ms)
				450 * time.Millisecond: 2,
				750 * time.Millisecond: 2,
				// from call at 600ms (+200ms wait = 800ms)
				850 * time.Millisecond: 3,
				// still 3 at at the end
				1350 * time.Millisecond: 3,
			},
		},
		{
			name:    "many calls, one cancel, four triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []testOp{
				{delay: 100 * time.Millisecond}, // trigger 1
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond}, // trigger 2 delayed to 500ms
				{delay: 600 * time.Millisecond}, // skipped, too close to 500ms
				{delay: 700 * time.Millisecond},
				{delay: 800 * time.Millisecond, cancel: true},
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond: 0,
				// call at 100ms
				150 * time.Millisecond: 1,
				450 * time.Millisecond: 1,
				// from call at 300ms (+200ms wait = 500ms)
				550 * time.Millisecond: 2,
				750 * time.Millisecond: 2,
				// canceled at 800ms
				850 * time.Millisecond: 2,
				// still 2 at at the end
				1050 * time.Millisecond: 2,
			},
		},
		{
			name:    "many calls, one cancel, four triggers",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []testOp{
				{delay: 100 * time.Millisecond}, // trigger 1
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond}, // trigger 2 delayed to 500ms
				{delay: 600 * time.Millisecond}, // skipped, too close to 500ms
				{delay: 700 * time.Millisecond},
				{delay: 725 * time.Millisecond, cancel: true},
				{delay: 800 * time.Millisecond}, // skipped, too close to 700ms
				{delay: 900 * time.Millisecond},
				{delay: 900 * time.Millisecond},
				{delay: 1000 * time.Millisecond}, // trigger 4 delayed to 1200ms
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond: 0,
				// call at 100ms
				150 * time.Millisecond: 1,
				450 * time.Millisecond: 1,
				// from call at 300ms (+200ms wait = 500ms)
				550 * time.Millisecond:  2,
				750 * time.Millisecond:  2,
				850 * time.Millisecond:  2,
				1150 * time.Millisecond: 2,
				// from call at 1000ms (+200ms wait = 1200ms)
				1250 * time.Millisecond: 3,
				// still 4 at at the end
				1750 * time.Millisecond: 3,
			},
		},
		{
			name:    "many triggers within wait time",
			wait:    200 * time.Millisecond,
			options: []Option{Leading(), Trailing()},
			calls: []testOp{
				{delay: 100 * time.Millisecond}, // trigger 1
				{delay: 200 * time.Millisecond},
				{delay: 300 * time.Millisecond},
				{delay: 400 * time.Millisecond},
				{delay: 500 * time.Millisecond},
				{delay: 600 * time.Millisecond},
				{delay: 700 * time.Millisecond}, // trigger 2 at 900ms
			},
			wantTriggers: map[time.Duration]int{
				50 * time.Millisecond:  0,
				150 * time.Millisecond: 1,
				850 * time.Millisecond: 1,
				950 * time.Millisecond: 2,
				// still 2 at at the end
				1450 * time.Millisecond: 2,
			},
		},
	}

	runTestCases(t, tests)
}
