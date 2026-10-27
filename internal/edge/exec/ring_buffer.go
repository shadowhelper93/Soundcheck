package exec

import "sync"

type ringBuffer struct {
	mu   sync.RWMutex
	data []byte
	cap  int
	pos  int
	full bool
}

func newRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{data: make([]byte, capacity), cap: capacity}
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(p)
	if r.cap <= 0 || n == 0 {
		return n, nil
	}
	if n >= r.cap {
		copy(r.data, p[n-r.cap:])
		r.pos = 0
		r.full = true
		return n, nil
	}
	first := min(n, r.cap-r.pos)
	copy(r.data[r.pos:], p[:first])
	if first < n {
		copy(r.data, p[first:])
		r.pos = n - first
		r.full = true
		return n, nil
	}
	r.pos += first
	if r.pos >= r.cap {
		r.pos = 0
		r.full = true
	}
	return n, nil
}

func (r *ringBuffer) Bytes() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.cap <= 0 {
		return nil
	}
	if !r.full {
		out := make([]byte, r.pos)
		copy(out, r.data[:r.pos])
		return out
	}
	out := make([]byte, r.cap)
	copy(out, r.data[r.pos:])
	copy(out[r.cap-r.pos:], r.data[:r.pos])
	return out
}
