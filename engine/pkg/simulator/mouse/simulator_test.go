package mouse

import "testing"

func TestScalePath(t *testing.T) {
	actions := []Action{
		{Type: TypePress, X: 0, Y: 0},
		{Type: TypeMove, X: 100, Y: 0},
		{Type: TypeRelease, X: 100, Y: 0},
	}
	out := ScalePath(actions, 50)
	if out[2].X != 50 {
		t.Fatalf("expected release x=50, got %d", out[2].X)
	}
}
