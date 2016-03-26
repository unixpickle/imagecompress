package smallbasis

import (
	"math"

	"github.com/unixpickle/num-analysis/kahan"
	"github.com/unixpickle/num-analysis/ludecomp"
)

func linearCombination(vecs []ludecomp.Vector, coeffs []float64) []float64 {
	if len(vecs) == 0 {
		return nil
	}

	res := make([]float64, len(vecs[0]))
	tempSumList := make([]float64, len(coeffs))

	for comp := range res {
		for i, vec := range vecs {
			tempSumList[i] = coeffs[i] * vec[comp]
		}
		res[comp] = kahan.Sum64(tempSumList)
	}

	return res
}

func roundFloat(f float64) int {
	return int(math.Floor(f + 0.5))
}
