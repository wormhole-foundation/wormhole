package common

import "golang.org/x/exp/constraints"

func Min[K constraints.Ordered](v []K) *K {
	if len(v) == 0 {
		return nil
	}

	lowest := v[0]
	for _, k := range v {
		if k < lowest {
			lowest = k
		}
	}

	return &lowest
}
