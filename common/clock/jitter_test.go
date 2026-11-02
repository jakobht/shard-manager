package clock

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestClock_jitteredTimerNeverFiresEarly(t *testing.T) {
	fake := clockwork.NewFakeClock()
	ts := &clock{Clock: fake}

	const d = time.Second
	maxDelay := d + time.Duration(float64(d)*maxJitterFraction)
	timer := ts.NewJitteredTimer(d)
	defer timer.Stop()
	// Each Reset samples fresh jitter.
	for range 20 {
		fake.Advance(d - time.Nanosecond)
		require.False(t, fired(timer), "fired before the requested delay")
		fake.Advance(maxDelay - d + time.Nanosecond)
		require.True(t, fired(timer), "did not fire within the jitter bound")
		timer.Reset(d)
	}
}

func fired(timer Timer) bool {
	select {
	case <-timer.Chan():
		return true
	default:
		return false
	}
}
