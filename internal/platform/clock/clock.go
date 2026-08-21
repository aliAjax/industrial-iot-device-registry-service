package clock

import "time"

type Clock interface {
	Now() time.Time
	NewTicker(d time.Duration) *time.Ticker
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

func (SystemClock) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

type FixedClock struct {
	At time.Time
}

func (c FixedClock) Now() time.Time {
	if c.At.IsZero() {
		return time.Unix(0, 0).UTC()
	}
	return c.At.UTC()
}

func (c FixedClock) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}
