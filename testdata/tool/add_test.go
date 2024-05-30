package add

import "testing"

func TestAdd(t *testing.T) {
	a := 1
	b := 2

	exp := 3
	got := add(a, b)

	if got != exp {
		t.Errorf("expected %v, got %v", exp, got)
	}
}
