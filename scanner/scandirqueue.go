package scanner

// Implementation adapted from github.com/eapache/queue:
//    The MIT License (MIT)
//    Copyright (c) 2014 Evan Huus

var nilscanDirState scanDirState

// ScanDirStateQueue represents a single instance of the queue data structure.
type ScanDirStateQueue struct {
	buf               []scanDirState
	head, tail, count int
	minlen            int
}

// NewScanDirStateQueue constructs and returns a new ScanDirStateQueue with an initial capacity.
func NewScanDirStateQueue(capacity int) *ScanDirStateQueue {
	// min capacity of 16
	if capacity < 16 {
		capacity = 16
	}
	return &ScanDirStateQueue{buf: make([]scanDirState, capacity), minlen: capacity}
}

// Len returns the number of elements currently stored in the queue.
func (q *ScanDirStateQueue) Len() int {
	return q.count
}

// Push puts an element on the end of the queue.
func (q *ScanDirStateQueue) Push(elem scanDirState) {
	if q.count == len(q.buf) {
		q.resize()
	}

	q.buf[q.tail] = elem
	q.tail = (q.tail + 1) % len(q.buf)
	q.count++
}

// Peek returns the element at the head of the queue. This call panics
// if the queue is empty.
func (q *ScanDirStateQueue) Peek() scanDirState {
	if q.Len() <= 0 {
		panic("queue: empty queue")
	}
	return q.buf[q.head]
}

// Get returns the element at index i in the queue. If the index is
// invalid, the call will panic.
func (q *ScanDirStateQueue) Get(i int) scanDirState {
	if i >= q.Len() || i < 0 {
		panic("queue: index out of range")
	}
	modi := (q.head + i) % len(q.buf)
	return q.buf[modi]
}

// Pop removes the element from the front of the queue.
// This call panics if the queue is empty.
func (q *ScanDirStateQueue) Pop() scanDirState {
	if q.Len() <= 0 {
		panic("queue: empty queue")
	}
	v := q.buf[q.head]
	// set to nil to avoid keeping reference to objects
	// that would otherwise be garbage collected
	q.buf[q.head] = nilscanDirState
	q.head = (q.head + 1) % len(q.buf)
	q.count--
	if len(q.buf) > q.minlen && q.count*4 <= len(q.buf) {
		q.resize()
	}
	return v
}

func (q *ScanDirStateQueue) resize() {
	newBuf := make([]scanDirState, q.count*2)

	if q.tail > q.head {
		copy(newBuf, q.buf[q.head:q.tail])
	} else {
		copy(newBuf, q.buf[q.head:len(q.buf)])
		copy(newBuf[len(q.buf)-q.head:], q.buf[:q.tail])
	}

	q.head = 0
	q.tail = q.count
	q.buf = newBuf
}
