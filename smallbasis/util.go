package smallbasis

import (
	"math"

	"github.com/unixpickle/num-analysis/kahan"
	"github.com/unixpickle/num-analysis/linalg"
)

func linearCombination(vecs []linalg.Vector, coeffs []float64) []float64 {
	if len(vecs) == 0 {
		return nil
	}

	res := make([]float64, len(vecs[0]))

	for comp := range res {
		s := kahan.NewSummer64()
		for i, vec := range vecs {
			s.Add(coeffs[i] * vec[comp])
		}
		res[comp] = s.Sum()
	}

	return res
}

func roundFloat(f float64) int {
	return int(math.Floor(f + 0.5))
}
