package engine

import (
	"sync"
)

type Barrier interface {
	Wait() // Wait is blocking if `Pause` called and non-blocking after `Resume` called
	Pause()
	Resume()
}

func newLockBarrier() Barrier { return &lockBarrier{} }

type lockBarrier struct{ sync.RWMutex }

func (lb *lockBarrier) Wait()   { lb.RLock(); lb.RUnlock() }
func (lb *lockBarrier) Pause()  { lb.Lock() }
func (lb *lockBarrier) Resume() { lb.Unlock() }
