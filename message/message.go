package message

type Message struct {
	ID       int    `json:"id"`
	FromName string `json:"from_name"`
	Body     string `json:"body"`
}

// SetNextID sets the ID to the next available ID.
func (m *Message) SetNextID() {
	nextMessageMu.Lock()
	defer nextMessageMu.Unlock()

	nextMessageID++
	m.ID = nextMessageID
}
