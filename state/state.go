package state

type State int

const (
	Grey   State = iota
	Green
	Yellow
	Red
)

func (s State) String() string {
	return [...]string{"grey", "green", "yellow", "red"}[s]
}

// Highest returns the highest-priority state. Priority: Red > Yellow > Green > Grey.
func Highest(states []State) State {
	best := Grey
	for _, s := range states {
		if s > best {
			best = s
		}
	}
	return best
}
