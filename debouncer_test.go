package debounce

import (
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getFuncName(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

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
			assert.NotNil(t, d.maxTimer)

			// Check function storage
			gotFnPtr := d.fn.Load()

			if tt.wantFnNil {
				assert.Nil(t, gotFnPtr)
			} else {
				require.NotNil(t, gotFnPtr)
				assert.NotNil(t, *gotFnPtr)
				assert.Equal(t, getFuncName(tt.fn), getFuncName(*gotFnPtr))
			}

			// Check initial state
			assert.False(t, d.dirty)
			assert.True(t, d.lastCall.IsZero())
			assert.True(t, d.lastInvoke.IsZero())
		})
	}
}
