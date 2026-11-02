package clock

import (
	"time"

	"math/rand/v2"
)

// maxJitterFraction is the most a wait is delayed, as a fraction of its duration.
const maxJitterFraction = 0.2

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	return d + time.Duration(rand.Int64N(int64(float64(d)*maxJitterFraction)+1))
}

type jitteredTimer struct{ Timer }

func (t jitteredTimer) Reset(d time.Duration) bool { return t.Timer.Reset(jitter(d)) }

type jitteredTicker struct{ Ticker }

func (t jitteredTicker) Reset(d time.Duration) { t.Ticker.Reset(jitter(d)) }
