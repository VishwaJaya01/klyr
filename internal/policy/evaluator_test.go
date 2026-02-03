package policy

import "testing"

func TestDecideAction(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		score      int
		threshold  int
		wantAction Action
		wantBlock  bool
	}{
		{"below-threshold", "enforce", 3, 5, ActionAllow, false},
		{"enforce-block", "enforce", 5, 5, ActionBlock, true},
		{"shadow", "shadow", 5, 5, ActionShadow, false},
		{"learn", "learn", 5, 5, ActionAllow, false},
		{"unknown", "unknown", 5, 5, ActionAllow, false},
	}

	for _, tt := range cases {
		action, block := DecideAction(tt.mode, tt.score, tt.threshold)
		if action != tt.wantAction || block != tt.wantBlock {
			t.Fatalf("%s: expected (%s,%v) got (%s,%v)", tt.name, tt.wantAction, tt.wantBlock, action, block)
		}
	}
}
