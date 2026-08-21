package backend

import (
	"testing"
)

func TestTagRingBuffer_PushAndGetAll(t *testing.T) {
	rb := NewTagRingBuffer(5)

	if rb.Size() != 0 {
		t.Errorf("expected size 0, got %d", rb.Size())
	}

	rb.Push(100, 1.0)
	rb.Push(200, 2.0)
	rb.Push(300, 3.0)

	if rb.Size() != 3 {
		t.Errorf("expected size 3, got %d", rb.Size())
	}

	pts := rb.GetAll()
	if len(pts) != 3 {
		t.Fatalf("expected 3 points, got %d", len(pts))
	}
	if pts[0].Timestamp != 100 || pts[0].Value != 1.0 {
		t.Errorf("expected point 0 to be (100, 1.0), got (%d, %f)", pts[0].Timestamp, pts[0].Value)
	}
	if pts[2].Timestamp != 300 || pts[2].Value != 3.0 {
		t.Errorf("expected point 2 to be (300, 3.0), got (%d, %f)", pts[2].Timestamp, pts[2].Value)
	}

	// Test wrapping around ring capacity
	rb.Push(400, 4.0)
	rb.Push(500, 5.0)
	rb.Push(600, 6.0) // overrides 100
	rb.Push(700, 7.0) // overrides 200

	if rb.Size() != 5 {
		t.Errorf("expected size 5 after wrap, got %d", rb.Size())
	}

	wrapped := rb.GetAll()
	if len(wrapped) != 5 {
		t.Fatalf("expected 5 points after wrap, got %d", len(wrapped))
	}
	expected := []int64{300, 400, 500, 600, 700}
	for i, exp := range expected {
		if wrapped[i].Timestamp != exp {
			t.Errorf("expected wrapped[%d].Timestamp = %d, got %d", i, exp, wrapped[i].Timestamp)
		}
	}
}

func TestTagRingBuffer_GetRange(t *testing.T) {
	rb := NewTagRingBuffer(10)

	for i := 1; i <= 10; i++ {
		rb.Push(int64(i*1000), float64(i))
	}

	// Query middle subrange [3000, 7000]
	sub := rb.GetRange(3000, 7000)
	if len(sub) != 5 {
		t.Fatalf("expected 5 points in range [3000, 7000], got %d", len(sub))
	}
	if sub[0].Timestamp != 3000 || sub[4].Timestamp != 7000 {
		t.Errorf("expected range [3000, 7000], got [%d, %d]", sub[0].Timestamp, sub[4].Timestamp)
	}

	// Query out of range
	empty := rb.GetRange(15000, 20000)
	if len(empty) != 0 {
		t.Errorf("expected 0 points, got %d", len(empty))
	}

	// Wrap around buffer and test GetRange
	for i := 11; i <= 15; i++ {
		rb.Push(int64(i*1000), float64(i))
	}
	// Buffer now has 6000..15000
	subWrapped := rb.GetRange(7000, 12000)
	if len(subWrapped) != 6 {
		t.Fatalf("expected 6 points in wrapped range [7000, 12000], got %d", len(subWrapped))
	}
	if subWrapped[0].Timestamp != 7000 || subWrapped[len(subWrapped)-1].Timestamp != 12000 {
		t.Errorf("expected [%d, %d], got [%d, %d]", 7000, 12000, subWrapped[0].Timestamp, subWrapped[len(subWrapped)-1].Timestamp)
	}
}

func TestTagRingBuffer_Clear(t *testing.T) {
	rb := NewTagRingBuffer(5)
	rb.Push(100, 1.0)
	rb.Push(200, 2.0)
	rb.Clear()

	if rb.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", rb.Size())
	}
	if len(rb.GetAll()) != 0 {
		t.Errorf("expected 0 items from GetAll after clear")
	}
	if len(rb.GetRange(0, 500)) != 0 {
		t.Errorf("expected 0 items from GetRange after clear")
	}
}

func TestTagRingBuffer_DynamicGrowth(t *testing.T) {
	// 1. Large capacity ring buffer starts with small initial allocation
	rb := NewTagRingBuffer(500000)
	if rb.AllocatedCapacity() != 1024 {
		t.Errorf("expected initial allocated capacity 1024, got %d", rb.AllocatedCapacity())
	}
	if rb.Capacity() != 500000 {
		t.Errorf("expected max capacity 500000, got %d", rb.Capacity())
	}

	// Push 1024 items
	for i := 1; i <= 1024; i++ {
		rb.Push(int64(i*10), float64(i))
	}
	if rb.Size() != 1024 {
		t.Errorf("expected size 1024, got %d", rb.Size())
	}
	if rb.AllocatedCapacity() != 1024 {
		t.Errorf("expected allocated capacity 1024, got %d", rb.AllocatedCapacity())
	}

	// Push 1025th item -> should dynamically grow to 2048
	rb.Push(10250, 1025.0)
	if rb.Size() != 1025 {
		t.Errorf("expected size 1025, got %d", rb.Size())
	}
	if rb.AllocatedCapacity() != 2048 {
		t.Errorf("expected allocated capacity 2048 after growth, got %d", rb.AllocatedCapacity())
	}

	// Verify all 1025 items preserved in correct order
	all := rb.GetAll()
	if len(all) != 1025 {
		t.Fatalf("expected 1025 items, got %d", len(all))
	}
	if all[0].Timestamp != 10 || all[0].Value != 1.0 {
		t.Errorf("expected first item (10, 1.0), got (%d, %f)", all[0].Timestamp, all[0].Value)
	}
	if all[1024].Timestamp != 10250 || all[1024].Value != 1025.0 {
		t.Errorf("expected last item (10250, 1025.0), got (%d, %f)", all[1024].Timestamp, all[1024].Value)
	}

	// Verify GetRange across the growth boundary
	rng := rb.GetRange(10200, 10250)
	if len(rng) != 6 {
		t.Fatalf("expected 6 items in range [10200, 10250], got %d", len(rng))
	}
	if rng[0].Timestamp != 10200 || rng[5].Timestamp != 10250 {
		t.Errorf("expected range [10200, 10250], got [%d, %d]", rng[0].Timestamp, rng[5].Timestamp)
	}

	// 2. Small capacity buffer (e.g. 50 items < 1024)
	small := NewTagRingBuffer(50)
	if small.AllocatedCapacity() != 50 {
		t.Errorf("expected initial allocated capacity 50, got %d", small.AllocatedCapacity())
	}
	for i := 1; i <= 60; i++ {
		small.Push(int64(i*10), float64(i))
	}
	if small.Size() != 50 {
		t.Errorf("expected size clamped to 50, got %d", small.Size())
	}
	if small.AllocatedCapacity() != 50 {
		t.Errorf("expected allocated capacity to stay 50, got %d", small.AllocatedCapacity())
	}
	smallAll := small.GetAll()
	if len(smallAll) != 50 || smallAll[0].Timestamp != 110 || smallAll[49].Timestamp != 600 {
		t.Errorf("expected small buffer items from 110 to 600, got first=%d, last=%d", smallAll[0].Timestamp, smallAll[49].Timestamp)
	}
}

