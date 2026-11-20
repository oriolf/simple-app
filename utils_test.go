package app

import "testing"

func TestMax(t *testing.T) {
	a, b := 5, 4
	max := Max([]int{a, b}, func(x, y int) bool { return x > y })
	if max != a {
		t.Errorf("Expected max %d, got %d", a, b)
	}
}

func TestMap(t *testing.T) {
	out := Map([]int{2}, func(x int) int { return 2 * x })
	if out[0] != 4 {
		t.Errorf("Expected %d, got %d", 4, out[0])
	}
}

func TestFilter(t *testing.T) {
	out := Filter([]int{1, 2, 3, 4, 5}, func(x int) bool { return x%2 == 0 })
	if len(out) != 2 {
		t.Errorf("Expected %d elements, got %d", 2, len(out))
	}
}
