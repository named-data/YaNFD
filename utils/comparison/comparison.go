package comparison

import "golang.org/x/exp/constraints"

func Min[V constraints.Ordered](a, b V) V {
	if a < b {
		return a
	} else {
		return b
	}
}

func Max[V constraints.Ordered](a, b V) V {
	if a > b {
		return a
	} else {
		return b
	}
}
