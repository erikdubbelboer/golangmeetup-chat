package message

import (
	"sync"
)

var (
	nextMessageID int
	nextMessageMu sync.Mutex
)

// SetNextMessageID sets the global next available ID.
func SetNextMessageID(id int) {
	nextMessageMu.Lock()
	defer nextMessageMu.Unlock()

	nextMessageID = id
}
