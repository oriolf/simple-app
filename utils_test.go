package app

import "testing"

func TestMax(t *testing.T) {
	a, b := 5, 4
	max := Max([]int{a, b}, func(x, y int) bool { return x > y })
	if max != a {
		t.Errorf("Expected max %d, got %d", a, b)
	}
}
