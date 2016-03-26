package smallbasis

import (
	"math"

	"github.com/unixpickle/num-analysis/ludecomp"
)

// BasisMatrix generates a column matrix for
// the standard image basis elements.
func BasisMatrix(size int) *ludecomp.Matrix {
	res := ludecomp.NewMatrix(size)
	for i := 0; i < size/2; i++ {
		freq := float64(i+1) * 2 * math.Pi / float64(size)
		for j := 0; j < size; j++ {
			argument := float64(j)
			res.Set(j, 2*i, math.Cos(argument*freq))
			res.Set(j, 2*i+1, math.Sin(argument*freq))
		}
	}

	// The last sin() should be replaced with cos(0*N) in every case,
	// since the basis needs a vector of all 1's.
	for i := 0; i < size; i++ {
		res.Set(i, size-1, 1)
	}

	normalizeColumns(res)

	return res
}

func normalizeColumns(m *ludecomp.Matrix) {
	vec := make(ludecomp.Vector, m.N)
	for col := 0; col < m.N; col++ {
		for row := 0; row < m.N; row++ {
			vec[row] = m.Get(row, col)
		}
		invMag := 1.0 / math.Sqrt(vec.Dot(vec))
		for row := 0; row < m.N; row++ {
			m.Set(row, col, vec[row]*invMag)
		}
	}
}
