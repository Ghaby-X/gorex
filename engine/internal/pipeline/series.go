package pipeline

// Series is a read-only view of recent values, newest first.
type Series interface {
	// Len is the number of values held (at most the capacity).
	Len() int
	// At returns the value i steps back; At(0) is the latest.
	At(i int) float64
}

// Rolling is a fixed-capacity ring buffer of float64 values.
type Rolling struct {
	buf   []float64
	next  int
	count int
}

// NewRolling returns a buffer holding at most capacity values.
func NewRolling(capacity int) *Rolling {
	if capacity < 1 {
		capacity = 1
	}
	return &Rolling{buf: make([]float64, capacity)}
}

// Push appends a value, evicting the oldest when full.
func (r *Rolling) Push(v float64) {
	r.buf[r.next] = v
	r.next = (r.next + 1) % len(r.buf)
	if r.count < len(r.buf) {
		r.count++
	}
}

func (r *Rolling) Len() int { return r.count }

// Cap is the maximum number of values held.
func (r *Rolling) Cap() int { return len(r.buf) }

// Full reports whether the buffer has reached capacity.
func (r *Rolling) Full() bool { return r.count == len(r.buf) }

// At returns the value i steps back. It panics if i is out of range.
func (r *Rolling) At(i int) float64 {
	if i < 0 || i >= r.count {
		panic("pipeline: Rolling.At index out of range")
	}
	idx := (r.next - 1 - i + 2*len(r.buf)) % len(r.buf)
	return r.buf[idx]
}
