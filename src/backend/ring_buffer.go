package backend

import (
	"sort"
	"sync"
)

const defaultInitialRingCap = 1024

// SamplePoint represents a single recorded data point in history
type SamplePoint struct {
	Timestamp int64   `json:"t"` // Unix timestamp in milliseconds
	Value     float64 `json:"v"` // Numeric value for chart/analysis
}

// TagRingBuffer is a thread-safe circular buffer with dynamic growth up to a maximum capacity
type TagRingBuffer struct {
	mu       sync.RWMutex
	capacity int           // maximum allowed sample capacity
	data     []SamplePoint // dynamically grown backing slice
	head     int           // points to the next write slot
	size     int           // number of items currently in buffer
}

// NewTagRingBuffer creates a new ring buffer with the given maximum capacity.
// Memory is allocated dynamically on demand up to capacity, starting at 1024 items.
func NewTagRingBuffer(capacity int) *TagRingBuffer {
	if capacity <= 0 {
		capacity = 500000 // default 500,000 points per tag
	}
	initCap := defaultInitialRingCap
	if initCap > capacity {
		initCap = capacity
	}
	if initCap <= 0 {
		initCap = 1
	}
	return &TagRingBuffer{
		capacity: capacity,
		data:     make([]SamplePoint, initCap),
	}
}

// Push adds a new point to the ring buffer in O(1) amortized time, dynamically growing memory if needed
func (rb *TagRingBuffer) Push(timestamp int64, value float64) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	currentCap := len(rb.data)

	// If buffer is full to its current slice length but hasn't reached max capacity yet, grow it
	if rb.size == currentCap && currentCap < rb.capacity {
		newCap := currentCap * 2
		if newCap > rb.capacity {
			newCap = rb.capacity
		}
		newData := make([]SamplePoint, newCap)
		headIdx := (rb.head - rb.size + currentCap) % currentCap
		for i := 0; i < rb.size; i++ {
			newData[i] = rb.data[(headIdx+i)%currentCap]
		}
		rb.data = newData
		rb.head = rb.size
		currentCap = newCap
	}

	rb.data[rb.head] = SamplePoint{
		Timestamp: timestamp,
		Value:     value,
	}
	rb.head = (rb.head + 1) % currentCap
	if rb.size < rb.capacity {
		rb.size++
	}
}

// Clear empties the ring buffer
func (rb *TagRingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.head = 0
	rb.size = 0
}

// Size returns the current number of stored points
func (rb *TagRingBuffer) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// AllocatedCapacity returns the current dynamically allocated slice length
func (rb *TagRingBuffer) AllocatedCapacity() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return len(rb.data)
}

// Capacity returns the maximum allowed capacity
func (rb *TagRingBuffer) Capacity() int {
	return rb.capacity
}

// GetRange returns all points with timestamp in [startMs, endMs] in chronological order using binary search
func (rb *TagRingBuffer) GetRange(startMs, endMs int64) []SamplePoint {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 || startMs > endMs {
		return []SamplePoint{}
	}

	currentCap := len(rb.data)
	headIdx := (rb.head - rb.size + currentCap) % currentCap

	// Binary search for first index where timestamp >= startMs
	low := sort.Search(rb.size, func(i int) bool {
		return rb.data[(headIdx+i)%currentCap].Timestamp >= startMs
	})

	// Binary search for first index where timestamp > endMs
	high := sort.Search(rb.size, func(i int) bool {
		return rb.data[(headIdx+i)%currentCap].Timestamp > endMs
	})

	if low >= high {
		return []SamplePoint{}
	}

	count := high - low
	result := make([]SamplePoint, count)
	for i := 0; i < count; i++ {
		result[i] = rb.data[(headIdx+low+i)%currentCap]
	}
	return result
}

// GetAll returns all stored points in chronological order
func (rb *TagRingBuffer) GetAll() []SamplePoint {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return []SamplePoint{}
	}
	currentCap := len(rb.data)
	result := make([]SamplePoint, rb.size)
	headIdx := (rb.head - rb.size + currentCap) % currentCap
	for i := 0; i < rb.size; i++ {
		result[i] = rb.data[(headIdx+i)%currentCap]
	}
	return result
}
