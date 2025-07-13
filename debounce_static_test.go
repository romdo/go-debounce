package debounce

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

	runTestCases(t, trailingTestCases)
}

func TestNew_withTrailing(t *testing.T) {
	t.Parallel()

	// Explicitly add trailing option to all test cases. It is the default
	// behavior, but here we specifically check for when the option is used.
	tests := make([]testCase, 0, len(trailingTestCases))
	for _, tc := range trailingTestCases {
		tc.options = append(tc.options, Trailing())
		tests = append(tests, tc)
	}

	runTestCases(t, tests)
}

func TestNew_withLeading(t *testing.T) {
	t.Parallel()

	runTestCases(t, leadingTestCases)
}

func TestNew_withLeadingAndTrailing(t *testing.T) {
	t.Parallel()

	runTestCases(t, leadingAndTrailingTestCases)
}

func TestNew_withMaxWait(t *testing.T) {
	t.Parallel()

	runTestCases(t, maxWaitTestCases)
}

func TestNew_withMaxWaitAndTrailing(t *testing.T) {
	t.Parallel()

	// Explicitly add trailing option to all test cases. It is the default
	// behavior, but here we specifically check for when the option is used.
	tests := make([]testCase, 0, len(maxWaitTestCases))
	for _, tc := range maxWaitTestCases {
		tc.options = append(tc.options, Trailing())
		tests = append(tests, tc)
	}

	runTestCases(t, tests)
}

func TestNew_withMaxWaitAndLeading(t *testing.T) {
	t.Parallel()

	runTestCases(t, maxWaitAndLeadingTestCases)
}

func TestNew_withMaxWaitLeadingAndTrailing(t *testing.T) {
	t.Parallel()

	runTestCases(t, maxWaitLeadingAndTrailingTestCases)
}

// MARK: Trailing

var trailingTestCases = []testCase{
	{
		name: "zero wait duration, immediate trigger",
		wait: 0,
		calls: []int64{
			0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			0:  0,
			1:  50,
			2:  100,
			3:  150,
			4:  200,
			5:  250,
			6:  300,
			7:  350,
			8:  400,
			9:  450,
			10: 500,
		},
	},
	{
		name: "negative wait duration, immediate trigger",
		wait: -100 * time.Millisecond,
		calls: []int64{
			0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			0:  0,
			1:  50,
			2:  100,
			3:  150,
			4:  200,
			5:  250,
			6:  300,
			7:  350,
			8:  400,
			9:  450,
			10: 500,
		},
	},
	{
		name: "one call, one trigger",
		wait: 200 * time.Millisecond,
		calls: []int64{
			100,
		},
		wantMutable: map[int64]int64{
			0: 300, // Trailing trigger via call at 100 milliseconds.
		},
	},
	{
		name: "two calls, two triggers",
		wait: 200 * time.Millisecond,
		calls: []int64{
			100, 400,
		},
		wantMutable: map[int64]int64{
			0: 300, // Trailing trigger via call at 100 milliseconds.
			1: 600, // Trailing trigger via call at 400 milliseconds.
		},
	},
	{
		name: "one burst of calls, one trigger",
		wait: 200 * time.Millisecond,
		calls: []int64{
			100, 150, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			8: 700, // Trailing trigger via call at 500 milliseconds.
		},
	},
	{
		name: "one burst of parallel calls, one trigger",
		wait: 200 * time.Millisecond,
		calls: []int64{
			100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
		},
		want: []int64{
			300, // Trailing trigger via call at 100 milliseconds.
		},
	},
	{
		name: "one burst of calls ending with parallel calls, " +
			"one trigger",
		wait: 200 * time.Millisecond,
		calls: []int64{
			100, 150, 200, 250, 300, 300, 300, 300, 300, 300, 300,
		},
		want: []int64{
			500, // Trailing trigger via call at 300 milliseconds.
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
		wantMutable: map[int64]int64{
			9: 800, // Trailing trigger via call at 600 milliseconds.
		},
	},
	{
		name: "two close bursts of calls, two triggers",
		wait: 200 * time.Millisecond,
		calls: []int64{
			100, 150, 200,
			500, 550, 600,
		},
		wantMutable: map[int64]int64{
			2: 400, // Trailing trigger via call at 200 milliseconds.
			5: 800, // Trailing trigger via call at 600 milliseconds.
		},
	},
	{
		name: "two close bursts of calls, longer wait, one trigger",
		wait: 400 * time.Millisecond,
		calls: []int64{
			100, 150, 200,
			500, 550, 600,
		},
		wantMutable: map[int64]int64{
			5: 1000, // Trailing trigger via call at 600 milliseconds.
		},
	},
	{
		name: "two bursts of calls, two triggers",
		wait: 200 * time.Millisecond,
		calls: []int64{
			100, 150, 200, 250, 300,
			800, 850, 900, 950, 1000,
		},
		wantMutable: map[int64]int64{
			4: 500,  // Trailing trigger via call at 300 milliseconds.
			9: 1200, // Trailing trigger via call at 1000 milliseconds.
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
		wantMutable: map[int64]int64{
			2: 400, // Trailing trigger via call at 200 milliseconds.
			5: 800, // Trailing trigger via call at 600 milliseconds.
		},
	},
}

// MARK: Leading

var leadingTestCases = []testCase{
	{
		name:    "zero wait duration, immediate trigger",
		wait:    0,
		options: []Option{Leading()},
		calls: []int64{
			0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			0:  0,
			1:  50,
			2:  100,
			3:  150,
			4:  200,
			5:  250,
			6:  300,
			7:  350,
			8:  400,
			9:  450,
			10: 500,
		},
	},
	{
		name:    "negative wait duration, immediate trigger",
		wait:    -100 * time.Millisecond,
		options: []Option{Leading()},
		calls: []int64{
			0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			0:  0,
			1:  50,
			2:  100,
			3:  150,
			4:  200,
			5:  250,
			6:  300,
			7:  350,
			8:  400,
			9:  450,
			10: 500,
		},
	},
	{
		name:    "one call, one trigger",
		wait:    200 * time.Millisecond,
		options: []Option{Leading()},
		calls: []int64{
			100,
		},
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
		},
	},
	{
		name:    "two calls, two triggers",
		wait:    200 * time.Millisecond,
		options: []Option{Leading()},
		calls: []int64{
			100, 400,
		},
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			1: 400, // Leading trigger at 400 milliseconds.
		},
	},
	{
		name:    "one burst of calls, one trigger",
		wait:    200 * time.Millisecond,
		options: []Option{Leading()},
		calls: []int64{
			100, 150, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			5: 400, // Leading trigger at 400 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			3: 500, // Leading trigger at 500 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			5: 600, // Leading trigger at 600 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			3: 500, // Leading trigger at 500 milliseconds.
		},
	},
}

// MARK: Leading and trailing

var leadingAndTrailingTestCases = []testCase{
	{
		name:    "zero wait duration, immediate trigger",
		wait:    0,
		options: []Option{Leading(), Trailing()},
		calls: []int64{
			0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			0:  0,
			1:  50,
			2:  100,
			3:  150,
			4:  200,
			5:  250,
			6:  300,
			7:  350,
			8:  400,
			9:  450,
			10: 500,
		},
	},
	{
		name:    "negative wait duration, immediate trigger",
		wait:    -100 * time.Millisecond,
		options: []Option{Leading(), Trailing()},
		calls: []int64{
			0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			0:  0,
			1:  50,
			2:  100,
			3:  150,
			4:  200,
			5:  250,
			6:  300,
			7:  350,
			8:  400,
			9:  450,
			10: 500,
		},
	},
	{
		name:    "one call, one trigger",
		wait:    200 * time.Millisecond,
		options: []Option{Leading(), Trailing()},
		calls: []int64{
			100,
		},
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
		},
	},
	{
		name:    "two calls, two triggers",
		wait:    200 * time.Millisecond,
		options: []Option{Leading(), Trailing()},
		calls: []int64{
			100, 400,
		},
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			1: 400, // Leading trigger at 400 milliseconds.
		},
	},
	{
		name:    "one burst of calls, two triggers",
		wait:    200 * time.Millisecond,
		options: []Option{Leading(), Trailing()},
		calls: []int64{
			100, 151, 200, 250, 300, 350, 400, 450, 500,
		},
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			8: 700, // Trailing trigger via call at 500 milliseconds.
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
			300, // Trailing trigger via call at 100 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
		},
		want: []int64{
			500, // Trailing trigger via call at 300 milliseconds.
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
		wantMutable: map[int64]int64{
			0:  100, // Leading trigger at 100 milliseconds.
			8:  500, // Leading trigger at 500 milliseconds.
			10: 800, // Trailing trigger via call at 600 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			2: 400, // Trailing trigger via call at 200 milliseconds.
			5: 800, // Trailing trigger via call at 600 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100,  // Leading trigger at 100 milliseconds.
			5: 1000, // Trailing trigger via call at 600 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100,  // Leading trigger at 100 milliseconds.
			4: 500,  // Trailing trigger via call at 300 milliseconds.
			5: 800,  // Leading trigger at 800 milliseconds.
			9: 1200, // Trailing trigger via call at 1000 milliseconds.
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
		wantMutable: map[int64]int64{
			0: 100, // Leading trigger at 100 milliseconds.
			2: 400, // Trailing trigger via call at 200 milliseconds.
			3: 500, // Leading trigger at 500 milliseconds.
			5: 800, // Trailing trigger via call at 600 milliseconds.
		},
	},
}

// MARK: Max wait

var maxWaitTestCases = []testCase{
	{
		name: "zero maxWait duration, maxWait is ignored",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(0),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450,
		},
		wantMutable: map[int64]int64{
			5: 650, // Trailing trigger via call at 450 milliseconds.
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
		wantMutable: map[int64]int64{
			5: 650, // Trailing trigger via call at 450 milliseconds.
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
		wantMutable: map[int64]int64{
			4: 900, // Trailing trigger via call at 400 milliseconds.
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
		wantMutable: map[int64]int64{
			4: 600, // Trailing trigger via call at 400 milliseconds.
		},
	},
	{
		name: "maxWait is slightly longer than wait duration",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(250 * time.Millisecond),
		},
		calls: []int64{
			0, 100, 200, 300, 400,
		},
		wantMutable: map[int64]int64{
			3: 300, // Max wait via call at 300 milliseconds.
			4: 600, // Trailing trigger via call at 400 milliseconds.
		},
	},
	{
		name: "one burst within wait time",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(500 * time.Millisecond),
		},
		calls: []int64{
			0, 50, 70, 70, 150,
		},
		wantMutable: map[int64]int64{
			4: 350, // Trailing trigger via call at 150 milliseconds.
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
		wantMutable: map[int64]int64{
			5: 650, // Trailing trigger via call at 450 milliseconds.
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
		wantMutable: map[int64]int64{
			6: 550, // Max wait via call at 550 milliseconds.
		},
	},
	{
		name: "one burst across two maxWaits and one trailing trigger",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
		},
		calls: []int64{
			50, 150, 250, 350, 450, 550, 650, 750, 850, 950, 1050, 1150,
		},
		wantMutable: map[int64]int64{
			5:  550,  // Max wait via call at 550 milliseconds.
			10: 1050, // Max wait via call at 1050 milliseconds.
			11: 1350, // Trailing trigger via call at 1150 milliseconds.
		},
	},
	{
		name: "two bursts with a maxWait in the first burst",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
		},
		calls: []int64{
			0, 100, 200, 300, 400, 500, 600,
			900, 1000, 1100,
		},
		wantMutable: map[int64]int64{
			5: 500,  // Max wait via call at 500 milliseconds.
			6: 800,  // Trailing trigger via call at 600 milliseconds.
			9: 1300, // Trailing trigger via call at 1100 milliseconds.
		},
	},
	{
		name: "two bursts with maxWaits, reset, and trailing trigger",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
		},
		calls: []int64{
			50, 150, 250, 350, 450, 550, 650, 750, 850,
			1550, 1650, 1750, 1850, 1950, 2050, 2150,
		},
		resets: []int64{
			950,
		},
		wantMutable: map[int64]int64{
			5:  550,  // Max wait via call at 550 milliseconds.
			14: 2050, // Max wait via call at 2050 milliseconds.
			15: 2350, // Trailing trigger via call at 2150 milliseconds.
		},
	},
}

// MARK: Max wait and leading

var maxWaitAndLeadingTestCases = []testCase{
	{
		name: "zero maxWait duration, maxWait is ignored",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(0),
			Leading(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450,
		},
		wantMutable: map[int64]int64{
			0: 0, // Leading trigger at 0 milliseconds.
		},
	},
	{
		name: "negative maxWait duration, maxWait is ignored",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(-100 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450,
		},
		wantMutable: map[int64]int64{
			0: 0, // Leading trigger at 0 milliseconds.
		},
	},
	{
		name: "maxWait shorter than wait duration, maxWait is ignored",
		wait: 500 * time.Millisecond,
		options: []Option{
			MaxWait(200 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			0, 100, 200, 300, 400,
		},
		wantMutable: map[int64]int64{
			0: 0, // Leading trigger at 0 milliseconds.
		},
	},
	{
		name: "maxWait equal to wait duration, maxWait is ignored",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(200 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			0, 100, 200, 300, 400,
		},
		wantMutable: map[int64]int64{
			0: 0, // Leading trigger at 0 milliseconds.
		},
	},
	{
		name: "maxWait is slightly longer than wait duration",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(250 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			0, 100, 200, 300, 400, 500, 600, 700,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			3: 300, // Leading max wait at 300 milliseconds.
			6: 600, // Leading max wait at 600 milliseconds.
		},
	},
	{
		name: "one burst within wait time",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(500 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			0, 50, 70, 70, 150,
		},
		wantMutable: map[int64]int64{
			0: 0, // Leading trigger at 0 milliseconds.
		},
	},
	{
		name: "one burst until right before maxWait",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(500 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450,
		},
		wantMutable: map[int64]int64{
			0: 0, // Leading trigger at 0 milliseconds.
		},
	},
	{
		name: "one burst until right after maxWait",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(500 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450, 550,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			6: 550, // Leading max wait at 550 milliseconds.
		},
	},
	{
		name: "one burst across two maxWaits",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(425 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			50, 150, 250, 350, 450, 550, 650, 750, 850, 950, 1050, 1150,
		},
		wantMutable: map[int64]int64{
			0:  50,   // Leading trigger at 50 milliseconds.
			5:  550,  // Leading max wait at 550 milliseconds.
			10: 1050, // Leading max wait at 1050 milliseconds.
		},
	},
	{
		name: "two bursts with a maxWait between them",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			0, 100, 200, 300, 400,
			550, 650, 750, 850,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 100 milliseconds.
			5: 550, // Max wait via call at 550 milliseconds.
		},
	},
	{
		name: "one burst with maxWaits and reset",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			50, 150, 250, 350, 450, 550, 650, 750, 850, 950, 1050, 1150,
			1250, 1350, 1450,
		},
		resets: []int64{
			775,
		},
		wantMutable: map[int64]int64{
			0:  50,   // Leading trigger at 50 milliseconds.
			5:  550,  // Leading max wait at 550 milliseconds.
			8:  850,  // Leading trigger at 850 milliseconds.
			13: 1350, // Leading max wait at 1350 milliseconds.
		},
	},
	{
		name: "two bursts with maxWaits and a reset",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
			Leading(),
		},
		calls: []int64{
			50, 150, 250, 350, 450, 550, 650, 750, 850,
			1550, 1650, 1750, 1850, 1950, 2050, 2150,
		},
		resets: []int64{
			675,
		},
		wantMutable: map[int64]int64{
			0:  50,   // Leading trigger at 50 milliseconds.
			5:  550,  // Leading max wait at 550 milliseconds.
			7:  750,  // Leading trigger at 750 milliseconds.
			9:  1550, // Leading trigger at 1550 milliseconds.
			14: 2050, // Leading max wait at 2050 milliseconds.
		},
	},
}

// MARK: Max wait, leading, and trailing

var maxWaitLeadingAndTrailingTestCases = []testCase{
	{
		name: "zero maxWait duration, maxWait is ignored",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(0),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			5: 650, // Trailing trigger via call at 450 milliseconds.
		},
	},
	{
		name: "negative maxWait duration, maxWait is ignored",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(-100 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			5: 650, // Trailing trigger via call at 450 milliseconds.
		},
	},
	{
		name: "maxWait shorter than wait duration, maxWait is ignored",
		wait: 500 * time.Millisecond,
		options: []Option{
			MaxWait(200 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 100, 200, 300, 400,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			4: 900, // Trailing trigger via call at 400 milliseconds.
		},
	},
	{
		name: "maxWait equal to wait duration, maxWait is ignored",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(200 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 100, 200, 300, 400,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			4: 600, // Trailing trigger via call at 400 milliseconds.
		},
	},
	{
		name: "maxWait is slightly longer than wait duration",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(250 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 100, 200, 300, 400, 500, 600, 700,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			3: 300, // Max wait via call at 300 milliseconds.
			6: 600, // Max wait via call at 600 milliseconds.
			7: 900, // Trailing trigger via call at 700 milliseconds.
		},
	},
	{
		name: "one burst within wait time",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(500 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 50, 70, 70, 150,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			4: 350, // Trailing trigger via call at 150 milliseconds.
		},
	},
	{
		name: "one burst until right before maxWait",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(500 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			5: 650, // Trailing trigger via call at 450 milliseconds.
		},
	},
	{
		name: "one burst until right after maxWait",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(500 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450, 550,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			6: 550, // Leading max wait at 550 milliseconds.
		},
	},
	{
		name: "one burst until right after maxWait, with trailing",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(500 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 50, 150, 250, 350, 450, 550, 650,
		},
		wantMutable: map[int64]int64{
			0: 0,   // Leading trigger at 0 milliseconds.
			6: 550, // Max wait via call at 550 milliseconds.
			7: 850, // Trailing trigger via call at 650 milliseconds.
		},
	},
	{
		name: "one burst across two maxWaits and one trailing trigger",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			50, 150, 250, 350, 450, 550, 650, 750, 850, 950, 1050, 1150,
		},
		wantMutable: map[int64]int64{
			0:  50,   // Leading trigger at 50 milliseconds.
			5:  550,  // Max wait via call at 550 milliseconds.
			10: 1050, // Max wait via call at 1050 milliseconds.
			11: 1350, // Trailing trigger via call at 1150 milliseconds.
		},
	},
	{
		name: "two bursts with a maxWait in the first burst",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 100, 200, 300, 400, 500, 600,
			900, 1000, 1100,
		},
		wantMutable: map[int64]int64{
			0: 0,    // Leading trigger at 0 milliseconds.
			5: 500,  // Max wait via call at 500 milliseconds.
			6: 800,  // Trailing trigger via call at 600 milliseconds.
			9: 1300, // Trailing trigger via call at 1100 milliseconds.
		},
	},
	{
		name: "two bursts with a maxWait between them",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			0, 100, 200, 300, 400,
			550, 650, 750, 850,
		},
		wantMutable: map[int64]int64{
			0: 0,    // Leading trigger at 100 milliseconds.
			5: 550,  // Max wait via call at 550 milliseconds.
			8: 1050, // Trailing trigger via call at 850 milliseconds.
		},
	},
	{
		name: "one burst with maxWaits and reset",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			50, 150, 250, 350, 450, 550, 650, 750, 850, 950, 1050, 1150,
			1250, 1350, 1450,
		},
		resets: []int64{
			775,
		},
		wantMutable: map[int64]int64{
			0:  50,   // Leading trigger at 50 milliseconds.
			5:  550,  // Max wait via call at 550 milliseconds.
			8:  850,  // Leading trigger at 850 milliseconds.
			13: 1350, // Max wait via call at 1350 milliseconds.
			14: 1650, // Trailing trigger via call at 1450 milliseconds.
		},
	},
	{
		name: "two bursts with maxWaits and a reset",
		wait: 200 * time.Millisecond,
		options: []Option{
			MaxWait(450 * time.Millisecond),
			Leading(),
			Trailing(),
		},
		calls: []int64{
			50, 150, 250, 350, 450, 550, 650, 750, 850,
			1550, 1650, 1750, 1850, 1950, 2050, 2150,
		},
		resets: []int64{
			675,
		},
		wantMutable: map[int64]int64{
			0:  50,   // Leading trigger at 50 milliseconds.
			5:  550,  // Max wait via call at 550 milliseconds.
			7:  750,  // Leading trigger at 750 milliseconds.
			8:  1050, // Trailing trigger via call at 850 milliseconds.
			9:  1550, // Leading trigger at 1550 milliseconds.
			14: 2050, // Max wait via call at 2050 milliseconds.
			15: 2350, // Trailing trigger via call at 2150 milliseconds.
		},
	},
}
