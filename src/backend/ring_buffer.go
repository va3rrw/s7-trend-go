package backend

import (
	"sort"
	"sync"
)

// SamplePoint represents a single recorded data point in history
type SamplePoint struct {
	Timestamp int64   `json:"t"` // Unix timestamp in milliseconds
	Value     float64 `json:"v"` // Numeric value for chart/analysis
}

// TagRingBuffer is a thread-safe circular buffer of fixed capacity for a single tag
type TagRingBuffer struct {
	mu       sync.RWMutex
	capacity int
	data     []SamplePoint
	head     int // points to the next write slot
	size     int // number of items currently in buffer
}

// NewTagRingBuffer creates a new ring buffer with the given capacity
func NewTagRingBuffer(capacity int) *TagRingBuffer {
	if capacity <= 0 {
		capacity = 500000 // default 500,000 points per tag
	}
	return &TagRingBuffer{
		capacity: capacity,
		data:     make([]SamplePoint, capacity),
	}
}

// Push adds a new point to the ring buffer in O(1) time
func (rb *TagRingBuffer) Push(timestamp int64, value float64) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.data[rb.head] = SamplePoint{
		Timestamp: timestamp,
		Value:     value,
	}
	rb.head = (rb.head + 1) % rb.capacity
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

// GetRange returns all points with timestamp in [startMs, endMs] in chronological order using binary search
func (rb *TagRingBuffer) GetRange(startMs, endMs int64) []SamplePoint {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 || startMs > endMs {
		return []SamplePoint{}
	}

	headIdx := (rb.head - rb.size + rb.capacity) % rb.capacity

	// Binary search for first index where timestamp >= startMs
	low := sort.Search(rb.size, func(i int) bool {
		return rb.data[(headIdx+i)%rb.capacity].Timestamp >= startMs
	})

	// Binary search for first index where timestamp > endMs
	high := sort.Search(rb.size, func(i int) bool {
		return rb.data[(headIdx+i)%rb.capacity].Timestamp > endMs
	})

	if low >= high {
		return []SamplePoint{}
	}

	count := high - low
	result := make([]SamplePoint, count)
	for i := 0; i < count; i++ {
		result[i] = rb.data[(headIdx+low+i)%rb.capacity]
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
	result := make([]SamplePoint, rb.size)
	headIdx := (rb.head - rb.size + rb.capacity) % rb.capacity
	for i := 0; i < rb.size; i++ {
		result[i] = rb.data[(headIdx+i)%rb.capacity]
	}
	return result
}
