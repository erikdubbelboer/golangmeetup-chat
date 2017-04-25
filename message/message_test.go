package message

import (
	"testing"
)

func TestSetNextID(t *testing.T) {
	var m Message

	m.SetNextID()
	id := m.ID
	m.SetNextID()
	if expected := id + 1; m.ID != expected {
		t.Fatalf("expected %d got %d", expected, m.ID)
	}
}

func BenchmarkSetNextID(b *testing.B) {
	var m Message
	for i := 0; i < b.N; i++ {
		m.SetNextID()
	}
}
