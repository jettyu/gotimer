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
	bucket_cnt int
	status     int32
	offset     []int64
	tickets    []time.Duration
	pre_base   []int64
	now        time.Duration
}

func NewTimeWheel(basetime time.Duration, intervals []int64) *TimeWheel {
	tw := &TimeWheel{}
	tw.bucket_cnt = len(intervals)
	tw.intervals = intervals

	tw.precisions = make([]time.Duration, tw.bucket_cnt)
	tw.pre_base = make([]int64, tw.bucket_cnt)
	tw.tickets = make([]time.Duration, tw.bucket_cnt)
	tw.precisions[0] = basetime
	for i := 0; i < tw.bucket_cnt; i++ {

		tw.precisions[i] = basetime
		tw.pre_base[i] = 1
		for j := 0; j < i; j++ {
			tw.precisions[i] *= time.Duration(tw.intervals[j])
			tw.pre_base[i] *= tw.intervals[j]
		}
		tw.tickets[i] = tw.precisions[i] * time.Duration(tw.intervals[i])
	}

	tw.curIndexs = make([]int64, tw.bucket_cnt)
	tw.offset = make([]int64, tw.bucket_cnt)

	tw.tasks = make([][]TaskList, tw.bucket_cnt)
	for i := 0; i < tw.bucket_cnt; i++ {
		tw.tasks[i] = make([]TaskList, tw.intervals[i])
	}
	tw.start()
	return tw
}

func (this *TimeWheel) After(d time.Duration) <-chan struct{} {
	var i = 0
	for i = 0; i < this.bucket_cnt-1; i++ {
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
	var c chan struct{} = nil
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
	var i = 0
	for i = 0; i < this.bucket_cnt-1; i++ {
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
	go func(elems *list.List, tw *TimeWheel, i int) {
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
				for i := 0; i < this.bucket_cnt; i++ {
					if tw.UpdateOffset(i) == 0 {
						go tw.onTimer(i)
					}
				}
			}
		}
	}(this)
}

func (this *TimeWheel) UpdateOffset(index int) int64 {
	i := (this.offset[index] + 1) % int64(this.pre_base[index])
	atomic.StoreInt64(&this.offset[index], i)
	return i
}

func (this *TimeWheel) Stop() {
	atomic.StoreInt32(&this.status, 1)
}
