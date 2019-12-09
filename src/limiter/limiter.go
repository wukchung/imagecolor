package limiter

import (
	"errors"
	"sync"
)

/*
	Since we are trying to use maximum of our resources we need to have a way to control
*/
type Limiter struct {
	max     int64
	current int64
	change  *sync.Cond
	lock    sync.Mutex
}

func New(max int64) *Limiter {
	return &Limiter{
		max:    max,
		change: sync.NewCond(new(sync.Mutex)),
	}
}

func (mm *Limiter) add(increase int64) bool {
	mm.lock.Lock()
	defer mm.lock.Unlock()
	if mm.current+increase >= mm.max {
		return false
	}
	mm.current += increase
	return true
}

func (mm *Limiter) Add(increase int64) {
	mm.change.L.Lock()
	defer mm.change.L.Unlock()

	for !mm.add(increase) {
		mm.change.Wait()
	}
}

func (mm *Limiter) Sub(sub int64) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	if mm.current < sub {
		// this means we are having some programmatic error and that is serious
		panic(errors.New("getting bellow what was acquired - should never happen"))
	}

	mm.current -= sub
	mm.change.Signal()
}
