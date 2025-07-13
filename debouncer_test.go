package debounce

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDebouncer(t *testing.T) {
	testFn := func() {}

	tests := []struct {
		name         string
		wait         time.Duration
		fn           func()
		opts         []Option
		wantLeading  bool
		wantTrailing bool
		wantMaxWait  time.Duration
		wantFnNil    bool
	}{
		{
			name:         "default configuration with function",
			wait:         100 * time.Millisecond,
			fn:           testFn,
			opts:         nil,
			wantLeading:  false,
			wantTrailing: true, // defaults to trailing
			wantMaxWait:  0,
			wantFnNil:    false,
		},
		{
			name:         "nil function",
			wait:         100 * time.Millisecond,
			fn:           nil,
			opts:         nil,
			wantLeading:  false,
			wantTrailing: true,
			wantMaxWait:  0,
			wantFnNil:    true,
		},
		{
			name:         "zero wait duration",
			wait:         0,
			fn:           testFn,
			opts:         nil,
			wantLeading:  false,
			wantTrailing: true,
			wantMaxWait:  0,
			wantFnNil:    false,
		},
		{
			name:         "negative wait duration",
			wait:         -100 * time.Millisecond,
			fn:           testFn,
			opts:         nil,
			wantLeading:  false,
			wantTrailing: true,
			wantMaxWait:  0,
			wantFnNil:    false,
		},
		{
			name:         "leading option only",
			wait:         100 * time.Millisecond,
			fn:           testFn,
			opts:         []Option{Leading()},
			wantLeading:  true,
			wantTrailing: false,
			wantMaxWait:  0,
			wantFnNil:    false,
		},
		{
			name:         "trailing option only",
			wait:         100 * time.Millisecond,
			fn:           testFn,
			opts:         []Option{Trailing()},
			wantLeading:  false,
			wantTrailing: true,
			wantMaxWait:  0,
			wantFnNil:    false,
		},
		{
			name:         "both leading and trailing",
			wait:         100 * time.Millisecond,
			fn:           testFn,
			opts:         []Option{Leading(), Trailing()},
			wantLeading:  true,
			wantTrailing: true,
			wantMaxWait:  0,
			wantFnNil:    false,
		},
		{
			name:         "maxWait option",
			wait:         100 * time.Millisecond,
			fn:           testFn,
			opts:         []Option{MaxWait(500 * time.Millisecond)},
			wantLeading:  false,
			wantTrailing: true,
			wantMaxWait:  500 * time.Millisecond,
			wantFnNil:    false,
		},
		{
			name:         "maxWait less than wait (should be disabled)",
			wait:         100 * time.Millisecond,
			fn:           testFn,
			opts:         []Option{MaxWait(50 * time.Millisecond)},
			wantLeading:  false,
			wantTrailing: true,
			wantMaxWait:  0, // should be disabled
			wantFnNil:    false,
		},
		{
			name:         "maxWait equal to wait (should be disabled)",
			wait:         100 * time.Millisecond,
			fn:           testFn,
			opts:         []Option{MaxWait(100 * time.Millisecond)},
			wantLeading:  false,
			wantTrailing: true,
			wantMaxWait:  0, // should be disabled
			wantFnNil:    false,
		},
		{
			name: "all options combined",
			wait: 100 * time.Millisecond,
			fn:   testFn,
			opts: []Option{
				Leading(),
				Trailing(),
				MaxWait(500 * time.Millisecond),
			},
			wantLeading:  true,
			wantTrailing: true,
			wantMaxWait:  500 * time.Millisecond,
			wantFnNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDebouncer(tt.wait, tt.fn, tt.opts...)

			require.NotNil(t, d)
			assert.Equal(t, tt.wait, d.wait)
			assert.Equal(t, tt.wantLeading, d.leading)
			assert.Equal(t, tt.wantTrailing, d.trailing)
			assert.Equal(t, tt.wantMaxWait, d.maxWait)

			// Check timer initialization
			assert.NotNil(t, d.timer)

			// Check function storage
			gotFn := d.fn

			if tt.wantFnNil {
				assert.Nil(t, gotFn)
			} else {
				require.NotNil(t, gotFn)
				assert.Equal(t, getFuncName(tt.fn), getFuncName(gotFn))
			}

			// Check initial state
			assert.False(t, d.dirty)
			assert.True(t, d.lastCall.IsZero())
			assert.True(t, d.lastInvoke.IsZero())
		})
	}
}

func TestDebouncer_Reset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wait     time.Duration
		setup    func(*Debouncer)
		validate func(*testing.T, *Debouncer)
	}{
		{
			name: "reset from initial state",
			wait: 100 * time.Millisecond,
		},
		{
			name: "reset after single debounce call",
			wait: 100 * time.Millisecond,
			setup: func(d *Debouncer) {
				d.Debounce()
			},
		},
		{
			name: "reset after multiple debounce calls",
			wait: 100 * time.Millisecond,
			setup: func(d *Debouncer) {
				d.Debounce()
				d.Debounce()
				d.Debounce()
			},
		},
		{
			name: "reset with different function set via DebounceWith",
			wait: 100 * time.Millisecond,
			setup: func(d *Debouncer) {
				newFn := func() {}
				d.DebounceWith(newFn)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var callCount int32
			fn := func() { atomic.AddInt32(&callCount, 1) }

			d := NewDebouncer(tt.wait, fn)

			// Setup the debouncer state
			if tt.setup != nil {
				tt.setup(d)
			}

			// Capture state before reset for comparison
			beforeFn := d.fn

			// Perform reset
			d.Reset()

			afterResetCount := atomic.LoadInt32(&callCount)

			// Verify reset cleared the expected state
			assert.False(t, d.dirty, "dirty should be false after reset")
			assert.True(t, d.lastCall.IsZero())
			assert.True(t, d.lastInvoke.IsZero())

			// Verify configuration is preserved
			assert.Equal(t, tt.wait, d.wait)

			// Verify function is preserved (Reset doesn't change the function)
			assert.Equal(t, getFuncName(beforeFn), getFuncName(d.fn))

			// Verify timers are still available (not nil)
			assert.NotNil(t, d.timer)

			// Run additional validation if provided
			if tt.validate != nil {
				tt.validate(t, d)
			}

			// Verify that the function was not invoked after reset.
			time.Sleep(tt.wait * 3)
			assert.Equal(t, afterResetCount, atomic.LoadInt32(&callCount))
		})
	}
}
