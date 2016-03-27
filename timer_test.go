package gotimer

import (
	"sync/atomic"
	"testing"
	"time"
)

func Test_AfterFunc(t *testing.T) {
	{
		i := int32(0)
		ch := make(chan bool)
		AfterFunc(time.Second*2, func() {
			atomic.AddInt32(&i, 1)
			ch <- true
		})
		<-ch
	}
}

func Test_After(t *testing.T) {
	select {
	case <-After(time.Second):
	}

	ResetPrecision(time.Millisecond*10, true)
	select {
	case <-After(time.Second * 3):
	}
	select {
	case <-After(time.Millisecond * 100):
	}
	ResetPrecision(time.Millisecond*100, true)
}

func Benchmark_AfterFunc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AfterFunc(time.Second, func() {})
	}
}

func Benchmark_After(b *testing.B) {
	for i := 0; i < b.N; i++ {
		After(time.Second)
	}
}
