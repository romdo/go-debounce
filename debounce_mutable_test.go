package debounce

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func makeMutableTestCases(testCases []testCase) []testCase {
	tests := make([]testCase, 0, len(testCases))
	for _, tc := range testCases {
		tc.mutable = true
		tests = append(tests, tc)
	}

	return tests
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

	runTestCases(t, makeMutableTestCases(trailingTestCases))
}

func TestNewMutable_withTrailing(t *testing.T) {
	t.Parallel()

	// Explicitly add trailing option to all test cases. It is the default
	// behavior, but here we specifically check for when the option is used.
	tests := make([]testCase, 0, len(trailingTestCases))
	for _, tc := range trailingTestCases {
		tc.options = append(tc.options, Trailing())
		tests = append(tests, tc)
	}

	runTestCases(t, makeMutableTestCases(tests))
}

func TestNewMutable_withLeading(t *testing.T) {
	t.Parallel()

	runTestCases(t, makeMutableTestCases(leadingTestCases))
}

func TestNewMutable_withLeadingAndTrailing(t *testing.T) {
	t.Parallel()

	runTestCases(t, makeMutableTestCases(leadingAndTrailingTestCases))
}

func TestNewMutable_withMaxWait(t *testing.T) {
	t.Parallel()

	runTestCases(t, makeMutableTestCases(maxWaitTestCases))
}

func TestNewMutable_withMaxWaitAndTrailing(t *testing.T) {
	t.Parallel()

	// Explicitly add trailing option to all test cases. It is the default
	// behavior, but here we specifically check for when the option is used.
	tests := make([]testCase, 0, len(maxWaitTestCases))
	for _, tc := range maxWaitTestCases {
		tc.options = append(tc.options, Trailing())
		tests = append(tests, tc)
	}

	runTestCases(t, makeMutableTestCases(tests))
}

func TestNewMutable_withMaxWaitAndLeading(t *testing.T) {
	t.Parallel()

	runTestCases(t, makeMutableTestCases(maxWaitAndLeadingTestCases))
}

func TestNewMutable_withMaxWaitLeadingAndTrailing(t *testing.T) {
	t.Parallel()

	runTestCases(t, makeMutableTestCases(maxWaitLeadingAndTrailingTestCases))
}
