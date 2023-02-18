package debounce

import (
	"time"
)

const longDelay = 24 * time.Hour

// stoppedTimer returns a stopped *time.Timer created with time.AfterFunc. The
// given function is not called until the timer is restarted with Reset.
func stoppedTimer(f func()) *time.Timer {
	t := time.AfterFunc(longDelay, f)
	t.Stop()

	return t
}
