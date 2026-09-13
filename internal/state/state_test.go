package state

import "testing"

// TestNewRunID_NoCollisions is a regression test for a real bug: the
// original implementation took the leading hex digits of
// time.Now().UnixNano(), which are the least likely to differ between
// calls microseconds apart. Two runs invoked back to back produced
// the same run ID. This test calls NewRunID in a tight loop -
// deliberately worse than real-world spacing - and fails if any two
// collide.
func TestNewRunID_NoCollisions(t *testing.T) {
	seen := make(map[string]bool)
	const n = 10000
	for i := 0; i < n; i++ {
		id := NewRunID()
		if seen[id] {
			t.Fatalf("collision on run id %q after %d iterations", id, i)
		}
		seen[id] = true
	}
}
