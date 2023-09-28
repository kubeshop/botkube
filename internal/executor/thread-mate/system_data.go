package thread_mate

import (
	"sync"
	"sync/atomic"
)

type (
	SystemData struct {
		sync.RWMutex
		dirty      bool
		roundRobin RoundRobin
	}
	RoundRobin struct {
		next uint32
		// TODO: rotationExclusion holds the assignee IDs as a key.
		// At a later stage, we can add for each entry the endDate, so it will be automatically enabled again.
		//rotationExclusion map[string]struct{}
	}
)

func (t *SystemData) ResetDirty() {
	t.Lock()
	defer t.Unlock()
	t.dirty = false
}

func (t *SystemData) MarkDirty() {
	t.Lock()
	defer t.Unlock()
	t.dirty = true
}

func (t *SystemData) IsDirty() bool {
	t.RLock()
	defer t.RUnlock()
	return t.dirty
}

func (t *SystemData) RoundRobinPickNext() uint32 {
	idx := atomic.AddUint32(&t.roundRobin.next, 1)
	t.MarkDirty()
	return idx
}
