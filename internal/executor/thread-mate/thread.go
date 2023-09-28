package thread_mate

import (
	"sync"
	"time"

	"github.com/kubeshop/botkube/pkg/api/executor"
)

type (
	// Thread represents a conversation thread.
	Thread struct {
		ID             string `csv:"-"`
		Assignee       Assignee
		MessageContext executor.Message `csv:"Message"`
		StartedAt      time.Time
		ResolvedBy     *Assignee
	}
	// Assignee represents a participant in a conversation.
	Assignee struct {
		ID          string
		DisplayName string
	}

	// Threads represents a collection of conversation threads.
	Threads struct {
		sync.RWMutex
		list  []Thread
		dirty bool
	}
)

// ResetDirty resets the dirty flag to indicate that changes have been saved.
func (t *Threads) ResetDirty() {
	t.Lock()
	defer t.Unlock()
	t.dirty = false
}

// IsDirty returns true if the Threads collection has been modified.
func (t *Threads) IsDirty() bool {
	t.RLock()
	defer t.RUnlock()
	return t.dirty
}

// Append adds a new thread to the Threads collection.
func (t *Threads) Append(th Thread) {
	t.Lock()
	defer t.Unlock()
	t.list = append(t.list, th)
	t.dirty = true
}

// Delete removes a thread with the specified ID from the Threads collection.
func (t *Threads) Delete(id string) *Thread {
	t.Lock()
	defer t.Unlock()

	for idx, item := range t.list {
		if item.ID != id {
			continue
		}

		t.list = append(t.list[:idx], t.list[idx+1:]...)
		t.dirty = true
		return &item
	}
	return nil
}

// Get returns a copy of the Threads collection.
func (t *Threads) Get() []Thread {
	t.RLock()
	defer t.RUnlock()
	return t.list
}

// Mutate applies a mutation function to a thread with the specified ID.
// If the thread is found, the mutation is applied, and the dirty flag is set to true.
func (t *Threads) Mutate(id string, mutate func(th *Thread)) bool {
	t.Lock()
	defer t.Unlock()

	for idx := range t.list {
		if t.list[idx].ID != id {
			continue
		}
		mutate(&t.list[idx])
		t.dirty = true
		return true
	}

	return false
}
