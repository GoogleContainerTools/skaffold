package archive

import (
	"hash"
	"sync"
)

// concurrentHasher wraps a hash.Hash so that writes to it
// happen on a separate go routine.
type concurrentHasher struct {
	hash    hash.Hash
	wg      sync.WaitGroup
	buffers chan []byte
}

func newConcurrentHasher(h hash.Hash) *concurrentHasher {
	ch := &concurrentHasher{
		hash:    h,
		buffers: make(chan []byte, 10),
	}

	go func() {
		for b := range ch.buffers {
			_, _ = ch.hash.Write(b)
			ch.wg.Done()
		}
	}()

	return ch
}

func (ch *concurrentHasher) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)

	ch.wg.Add(1)
	ch.buffers <- cp

	return len(p), nil
}

func (ch *concurrentHasher) Sum(b []byte) []byte {
	ch.wg.Wait()
	close(ch.buffers)
	return ch.hash.Sum(b)
}
