package gotimer

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type TaskNode struct {
	activeTime time.Duration
	task       func()
}

type TaskList struct {
	sync.Mutex
	Elems    *list.List
	c        chan struct{}
	cmap     map[time.Duration]chan struct{}
	cmapLock sync.Mutex
}

func (ml *TaskList) AddChan(d time.Duration) (c chan struct{}, f func()) {
	var ok bool
	ml.cmapLock.Lock()
	if ml.cmap == nil {
		ml.cmap = make(map[time.Duration]chan struct{})
		c = make(chan struct{})
		ml.cmap[d] = c
		f = func() { close(c) }
	} else {
		c, ok = ml.cmap[d]
		if !ok {
			c = make(chan struct{})
			ml.cmap[d] = c
			f = func() { close(c) }
		}
	}
	ml.cmapLock.Unlock()

	return c, f
}

func (ml *TaskList) AddTask(d time.Duration, f func()) {
	ml.Lock()
	if ml.Elems == nil {
		ml.Elems = list.New()
	}

	ml.Elems.PushBack(TaskNode{activeTime: d, task: f})
	ml.Unlock()
}

type TimeWheel struct {
	ticker     *time.Ticker
	tasks      [][]TaskList
	precisions []time.Duration
	intervals  []int64
	curIndexs  []int64
	bucketCnt  int
	status     int32
	offset     []int64
	tickets    []time.Duration
	preBase    []int64
	baseTime   time.Duration
}

//basetime is min precision.intervals the number of each bucket.
func NewTimeWheel(basetime time.Duration, intervals []int64) *TimeWheel {
	tw := &TimeWheel{}
	tw.baseTime = basetime
	tw.bucketCnt = len(intervals)
	tw.intervals = intervals

	tw.precisions = make([]time.Duration, tw.bucketCnt)
	tw.preBase = make([]int64, tw.bucketCnt)
	tw.tickets = make([]time.Duration, tw.bucketCnt)
	tw.precisions[0] = basetime
	for i := 0; i < tw.bucketCnt; i++ {

		tw.precisions[i] = basetime
		tw.preBase[i] = 1
		for j := 0; j < i; j++ {
			tw.precisions[i] *= time.Duration(tw.intervals[j])
			tw.preBase[i] *= tw.intervals[j]
		}
		tw.tickets[i] = tw.precisions[i] * time.Duration(tw.intervals[i])
	}

	tw.curIndexs = make([]int64, tw.bucketCnt)
	tw.offset = make([]int64, tw.bucketCnt)

	tw.tasks = make([][]TaskList, tw.bucketCnt)
	for i := 0; i < tw.bucketCnt; i++ {
		tw.tasks[i] = make([]TaskList, tw.intervals[i])
	}
	tw.start()
	return tw
}

func (this *TimeWheel) After(d time.Duration) <-chan struct{} {
	if d < this.baseTime {
		ch := make(chan struct{})
		time.AfterFunc(d, func() { close(ch) })
		return ch
	}
	var i = 0
	for i = 0; i < this.bucketCnt-1; i++ {
		if d < this.precisions[i+1] {
			break
		}
	}
	d += time.Duration(atomic.LoadInt64(&this.offset[i])) * this.precisions[0]
	d -= d % this.precisions[0]
	interval := int64(d / this.precisions[i])
	if interval > this.intervals[i] {
		panic(fmt.Errorf("TimeWheel wrong after time, interval=%d and aftertime=%d",
			this.intervals[i]*int64(this.precisions[i]), d))
	} else if interval == 0 && i == 0 {
		c := make(chan struct{})
		go func(c chan struct{}) {
			select {
			case <-time.After(d):
				close(c)
			}
		}(c)
		return c
	}

	index := (atomic.LoadInt64(&this.curIndexs[i]) + interval - 1) % this.intervals[i]
	ml := &this.tasks[i][index]
	var c chan struct{}
	if i != 0 {
		var f func()
		c, f = ml.AddChan(d)
		if f != nil {
			ml.AddTask(d, f)
		}
	} else {
		ml.Lock()
		if i == 0 {
			if ml.c == nil {
				ml.c = make(chan struct{})
			}
			c = ml.c
		}
		ml.Unlock()
	}

	return c
}

func (this *TimeWheel) AfterFunc(d time.Duration, f func()) {
	if d < this.baseTime {
		time.AfterFunc(d, f)
		return
	}
	var i = 0
	for i = 0; i < this.bucketCnt-1; i++ {
		if d < this.precisions[i+1] {
			break
		}
	}
	d += time.Duration(atomic.LoadInt64(&this.offset[i])) * this.precisions[0]
	interval := int64(d / this.precisions[i])
	if interval > this.intervals[i] {
		panic(fmt.Errorf("TimeWheel wrong after time, interval=%d and aftertime=%d",
			this.intervals[i]*int64(this.precisions[i]), d))
	} else if interval == 0 && i == 0 {
		go f()
	}

	index := (atomic.LoadInt64(&this.curIndexs[i]) + interval - 1) % this.intervals[i]
	ml := &this.tasks[i][index]
	ml.AddTask(d, f)
}

func (tw *TimeWheel) onTimer(i int) {
	curIndex := tw.curIndexs[i]
	atomic.StoreInt64(&tw.curIndexs[i], (curIndex+1)%tw.intervals[i])

	ml := &tw.tasks[i][curIndex]

	var elems *list.List
	var c chan struct{} = nil
	ml.Lock()
	c = ml.c
	ml.c = nil
	elems = ml.Elems
	ml.Elems = nil
	ml.Unlock()
	if c != nil {
		close(c)
	}
	if elems == nil {
		return
	}
	/*go*/ func(elems *list.List, tw *TimeWheel, i int) {
		e := elems.Front()
		if e != nil {
			for ; e != nil; e = e.Next() {
				tn := e.Value.(TaskNode)
				nextTime := tn.activeTime % tw.precisions[i]
				if nextTime == 0 ||
					i == 0 {
					go tn.task()
				} else {
					tw.AfterFunc(nextTime, tn.task)
				}
			}
		}
	}(elems, tw, i)
}
func (this *TimeWheel) start() {
	go func(tw *TimeWheel) {
		tw.ticker = time.NewTicker(tw.precisions[0])
		defer tw.ticker.Stop()
		for atomic.LoadInt32(&tw.status) == 0 {
			select {
			case <-tw.ticker.C:
				for i := 0; i < this.bucketCnt; i++ {
					if tw.UpdateOffset(i) == 0 {
						go tw.onTimer(i)
					}
				}
			}
		}
	}(this)
}

func (this *TimeWheel) UpdateOffset(index int) int64 {
	i := (this.offset[index] + 1) % int64(this.preBase[index])
	atomic.StoreInt64(&this.offset[index], i)
	return i
}

func (this *TimeWheel) Stop() {
	atomic.StoreInt32(&this.status, 1)
}
