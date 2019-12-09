package limiter

import (
	"log"
	"runtime"
	"sync"
	"time"
)

var memStat = &runtime.MemStats{}

/*
	As far I know there is not a good way to get precise current usage of memory of the go process and in the same time there is not possible to catch/recover from "out of memory" state.
	Therefore I created this memory checker to get at least some kind of control.
*/
type Memory struct {
	max    uint64
	change *sync.Cond
	lock   sync.Mutex
}

func NewMemory(max uint64) *Memory {
	m := &Memory{
		max:    max,
		change: sync.NewCond(new(sync.Mutex)),
	}

	go func() {
		// we want to make sure a deadlock is not created because of some GC delay
		for {
			time.Sleep(time.Second)
			m.CheckRelease()
		}
	}()

	return m
}

func (mm *Memory) CheckAddition(increase uint64) {
	mm.change.L.Lock()
	defer mm.change.L.Unlock()

	// get current taken memory and add the increase
	runtime.ReadMemStats(memStat)

	for memStat.Alloc+increase > mm.max {
		log.Println("Waiting for memory")
		mm.change.Wait()
		runtime.ReadMemStats(memStat)
	}
}

func (mm *Memory) CheckRelease() {
	mm.change.Signal()
}
