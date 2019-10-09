package gotimer

import (
	"sync/atomic"
	"time"
)

// XTimerHandler ...
type XTimerHandler struct {
	handers []*Timer
	buckets int32
	index   int32
}

// NewXTimerHandler ...
func NewXTimerHandler(precision time.Duration, buckets int) *XTimerHandler {
	xtw := &XTimerHandler{
		handers: make([]*Timer, buckets),
		buckets: int32(buckets),
	}
	for i := 0; i < buckets; i++ {
		xtw.handers[i] = NewTimer(precision)
	}
	return xtw
}

// After ...
func (p *XTimerHandler) After(d time.Duration) <-chan struct{} {
	index := atomic.AddInt32(&p.index, 1) % p.buckets
	return p.handers[index].After(d)
}

// AfterFunc ...
func (p *XTimerHandler) AfterFunc(d time.Duration, task func()) {
	index := atomic.AddInt32(&p.index, 1) % p.buckets
	p.handers[index].AfterFunc(d, task)
}

// Stop ...
func (p *XTimerHandler) Stop() {
	for _, v := range p.handers {
		v.Stop()
	}
}
