package gotimer

import (
	"time"
)

//1s = 100ms *10
//1min = 100ms * 100 * 6
//1hour = 3.60 * 100 * 100 * 100ms

var (
	// ElementCntPerBucket ...
	ElementCntPerBucket = []int64{256, 64, 64, 64, 64}
)

// Timer ...
type Timer struct {
	timerWheels *TimeWheel
	baseTime    time.Duration
}

func (p *Timer) init() {
	p.timerWheels = NewTimeWheel(p.baseTime, ElementCntPerBucket)
}

// NewTimer ...
func NewTimer(baseTime ...time.Duration) *Timer {
	var d time.Duration
	if len(baseTime) > 0 {
		d = baseTime[0]
	} else {
		d = DefaultPrecision
	}
	th := &Timer{
		baseTime: d,
	}
	th.init()
	return th
}

// AfterFunc ...
func (p *Timer) AfterFunc(d time.Duration, task func()) {
	p.timerWheels.AfterFunc(d, task)
}

// After ...
func (p *Timer) After(d time.Duration) <-chan struct{} {
	return p.timerWheels.After(d)
}

// Stop ...
func (p *Timer) Stop() {
	p.timerWheels.Stop()
}
