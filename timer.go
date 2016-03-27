package gotimer

import (
	"time"
)

var (
	GlobalPrecision = time.Millisecond * 100
	GlobalBaseMask  = 10
	globalxtimer    = NewXTimerHandler(GlobalPrecision, GlobalBaseMask)
)

func ResetPrecision(precision time.Duration, stop ...bool) {
	GlobalPrecision = precision
	oldxtimer := globalxtimer
	globalxtimer = NewXTimerHandler(GlobalPrecision, GlobalBaseMask)
	if len(stop) > 0 && stop[0] {
		oldxtimer.Stop()
	}
}

func ResetMask(mask int, stop ...bool) {
	GlobalBaseMask = mask
	oldxtimer := globalxtimer
	globalxtimer = NewXTimerHandler(GlobalPrecision, GlobalBaseMask)
	if len(stop) > 0 && stop[0] {
		oldxtimer.Stop()
	}
}

func ResetPrecisionAndMask(precision time.Duration, mask int, stop ...bool) {
	GlobalPrecision = precision
	GlobalBaseMask = mask
	oldxtimer := globalxtimer
	globalxtimer = NewXTimerHandler(GlobalPrecision, GlobalBaseMask)
	if len(stop) > 0 && stop[0] {
		oldxtimer.Stop()
	}
}

func After(d time.Duration) <-chan struct{} {
	return globalxtimer.After(d)
}

func AfterFunc(d time.Duration, f func()) {
	globalxtimer.AfterFunc(d, f)
}
