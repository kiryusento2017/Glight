package state

import "testing"

func TestHighest(t *testing.T) {
	cases := []struct {
		in   []State
		want State
	}{
		{nil, Grey},
		{[]State{Grey}, Grey},
		{[]State{Green, Grey}, Green},
		{[]State{Yellow, Green}, Yellow},
		{[]State{Red, Yellow, Green}, Red},
	}
	for _, c := range cases {
		if got := Highest(c.in); got != c.want {
			t.Errorf("Highest(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestString(t *testing.T) {
	if Grey.String() != "grey" {
		t.Fail()
	}
	if Green.String() != "green" {
		t.Fail()
	}
	if Yellow.String() != "yellow" {
		t.Fail()
	}
	if Red.String() != "red" {
		t.Fail()
	}
}
