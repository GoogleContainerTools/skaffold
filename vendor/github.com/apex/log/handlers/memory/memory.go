// Package memory implements an in-memory handler useful for testing, as the
// entries can be accessed after writes.
package memory

import (
	"sync"

	"github.com/apex/log"
)

// Handler implementation.
type Handler struct {
	mu      sync.Mutex
	Entries []*log.Entry
}

// New handler.
func New() *Handler {
	return &Handler{}
}

// HandleLog implements log.Handler.
func (h *Handler) HandleLog(e *log.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Entries = append(h.Entries, e)
	return nil
}
