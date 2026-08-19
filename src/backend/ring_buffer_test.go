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
