package gotimer

import (
	"time"
)

var (
	// DefaultPrecision ...
	DefaultPrecision = time.Millisecond * 100
	// DefaultBaseMask ...
	DefaultBaseMask = 10
	defaultXTimer   = NewXTimerHandler(DefaultPrecision, DefaultBaseMask)
)

// ResetPrecision ...
func ResetPrecision(precision time.Duration, stop ...bool) {
	DefaultPrecision = precision
	oldxtimer := defaultXTimer
	defaultXTimer = NewXTimerHandler(DefaultPrecision, DefaultBaseMask)
	if len(stop) > 0 && stop[0] {
		oldxtimer.Stop()
	}
}

// ResetMask ...
func ResetMask(mask int, stop ...bool) {
	DefaultBaseMask = mask
	oldxtimer := defaultXTimer
	defaultXTimer = NewXTimerHandler(DefaultPrecision, DefaultBaseMask)
	if len(stop) > 0 && stop[0] {
		oldxtimer.Stop()
	}
}

// ResetPrecisionAndMask ...
func ResetPrecisionAndMask(precision time.Duration, mask int, stop ...bool) {
	DefaultPrecision = precision
	DefaultBaseMask = mask
	oldxtimer := defaultXTimer
	defaultXTimer = NewXTimerHandler(DefaultPrecision, DefaultBaseMask)
	if len(stop) > 0 && stop[0] {
		oldxtimer.Stop()
	}
}

// After ...
func After(d time.Duration) <-chan struct{} {
	return defaultXTimer.After(d)
}

// AfterFunc ...
func AfterFunc(d time.Duration, f func()) {
	defaultXTimer.AfterFunc(d, f)
}
