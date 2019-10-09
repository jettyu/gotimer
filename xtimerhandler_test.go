package gotimer

import (
	"sync/atomic"
	"testing"
	"time"
)

func Test_XTimerHandlerAfterFunc(t *testing.T) {
	timer := NewXTimerHandler(DefaultPrecision, 10)
	defer timer.Stop()
	{
		i := int32(0)
		ch := make(chan bool)
		timer.AfterFunc(time.Second*2, func() {
			atomic.AddInt32(&i, 1)
			ch <- true
		})
		<-ch
	}
}

func Benchmark_XTimerHandlerAfterFunc(b *testing.B) {
	timer := NewXTimerHandler(DefaultPrecision, 256)
	defer timer.Stop()
	for i := 0; i < b.N; i++ {
		timer.AfterFunc(time.Second, func() {})
	}
}

func Benchmark_XTimerHandlerAfter(b *testing.B) {
	timer := NewXTimerHandler(DefaultPrecision, 256)
	defer timer.Stop()
	for i := 0; i < b.N; i++ {
		timer.After(time.Second)
	}
}
