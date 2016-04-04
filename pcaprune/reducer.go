package pcaprune

import (
	"encoding/binary"
	"io"
	"sort"

	"github.com/unixpickle/num-analysis/kahan"
	"github.com/unixpickle/num-analysis/linalg"
	"github.com/unixpickle/num-analysis/linalg/eigen"
	"github.com/unixpickle/num-analysis/linalg/leastsquares"
)

const maxEigenPrecision = 1e-6

type pcaReducer struct {
	solver *leastsquares.Solver
	basis  []linalg.Vector
}

func newPCAReducer(vecs []linalg.Vector, basisSize int) *pcaReducer {
	normalMat := linalg.NewMatrix(len(vecs[0]), len(vecs[0]))
	for i := 0; i < normalMat.Rows; i++ {
		for j := 0; j <= i; j++ {
			s := kahan.NewSummer64()
			for _, vec := range vecs {
				s.Add(vec[i] * vec[j])
			}
			normalMat.Set(i, j, s.Sum())
			normalMat.Set(j, i, s.Sum())
		}
	}
	vals, vecs := eigs(normalMat)
	sorter := &eigenSorter{vals: vals, vecs: vecs}
	sort.Sort(sorter)

	res := &pcaReducer{
		basis: make([]linalg.Vector, basisSize),
	}
	copy(res.basis, vecs)
	res.solver = leastsquares.NewSolver(matrixWithColumns(res.basis))

	return res
}

func (p *pcaReducer) Reduce(vec linalg.Vector) linalg.Vector {
	return p.solver.Solve(vec)
}

func (p *pcaReducer) WriteTo(w io.Writer) (int64, error) {
	var written int64

	if err := binary.Write(w, encodingEndian, uint32(len(p.basis))); err != nil {
		return written, err
	}
	written += 4

	if err := binary.Write(w, encodingEndian, uint32(len(p.basis[0]))); err != nil {
		return written, err
	}
	written += 4

	for _, vec := range p.basis {
		for _, val := range vec {
			if err := binary.Write(w, encodingEndian, float32(val)); err != nil {
				return written, err
			}
			written += 4
		}
	}
	return written, nil
}

func eigs(m *linalg.Matrix) ([]float64, []linalg.Vector) {
	// If we can get the answer up to maxEigenPrecision, it's good enough.
	// On the other hand, if we cannot, then we will have to wait until
	// the most accurate possible solution is found.
	res1 := eigen.SymmetricPrecAsync(m, maxEigenPrecision)
	res2 := eigen.SymmetricAsync(m)

	vals1 := make([]float64, 0, m.Rows)
	vals2 := make([]float64, 0, m.Rows)
	vecs1 := make([]linalg.Vector, 0, m.Rows)
	vecs2 := make([]linalg.Vector, 0, m.Rows)

	for {
		select {
		case val, ok := <-res1.Values:
			if !ok {
				close(res2.Cancel)
				return vals1, vecs1
			}
			vals1 = append(vals1, val)
			vecs1 = append(vecs1, <-res1.Vectors)
		case val, ok := <-res2.Values:
			if !ok {
				close(res1.Cancel)
				return vals2, vecs2
			}
			vals2 = append(vals2, val)
			vecs2 = append(vecs2, <-res2.Vectors)
		}
	}
}

func matrixWithColumns(c []linalg.Vector) *linalg.Matrix {
	res := linalg.NewMatrix(len(c[0]), len(c))
	for i := 0; i < res.Rows; i++ {
		for j := 0; j < res.Cols; j++ {
			res.Set(i, j, c[j][i])
		}
	}
	return res
}

type eigenSorter struct {
	vals []float64
	vecs []linalg.Vector
}

func (e *eigenSorter) Len() int {
	return len(e.vals)
}

func (e *eigenSorter) Less(i, j int) bool {
	return e.vals[i] > e.vals[j]
}

func (e *eigenSorter) Swap(i, j int) {
	e.vals[i], e.vals[j] = e.vals[j], e.vals[i]
	e.vecs[i], e.vecs[j] = e.vecs[j], e.vecs[i]
}
