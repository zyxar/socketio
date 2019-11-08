package engine

import (
	"sync"
)

type Resumer interface{ Resume() }
type Barrier interface {
	Wait() // Wait is blocking if `Pause` called and non-blocking after `Resume` called
	Pause() Resumer
}

func newLockBarrier() Barrier { return &lockBarrier{} }

type lockBarrier struct{ sync.RWMutex }

func (lb *lockBarrier) Wait()          { lb.RLock(); lb.RUnlock() }
func (lb *lockBarrier) Pause() Resumer { lb.Lock(); return lb }
func (lb *lockBarrier) Resume()        { lb.Unlock() }

func newWaitGroupBarrier() Barrier { return &waitGroupBarrier{} }

type waitGroupBarrier struct{ sync.WaitGroup }

func (wb *waitGroupBarrier) Pause() Resumer { wb.Add(1); return wb }
func (wb *waitGroupBarrier) Resume()        { wb.Done() }
