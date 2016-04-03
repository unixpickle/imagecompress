package pca

import (
	"sort"

	"github.com/unixpickle/num-analysis/kahan"
	"github.com/unixpickle/num-analysis/linalg"
	"github.com/unixpickle/num-analysis/linalg/eigen"
	"github.com/unixpickle/num-analysis/linalg/leastsquares"
)

type pcaReducer struct {
	solver *leastsquares.Solver
	basis  []linalg.Vector
}

func newPCAReducer(vecs []linalg.Vector, basisSize int) (*pcaReducer, error) {
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
	vals, vecs, err := eigen.InverseIteration(normalMat, 10000)
	if err != nil {
		return nil, err
	}
	sorter := &eigenSorter{vals: vals, vecs: vecs}
	sort.Sort(sorter)

	res := &pcaReducer{
		basis: make([]linalg.Vector, basisSize),
	}
	copy(res.basis, vecs)
	res.solver = leastsquares.NewSolver(matrixWithColumns(res.basis))

	return res, nil
}

func (p *pcaReducer) Reduce(vec linalg.Vector) linalg.Vector {
	return p.solver.Solve(vec)
}

type pcaExpander struct {
	basis []linalg.Vector
}

func (p *pcaExpander) Expand(vec linalg.Vector) linalg.Vector {
	res := make(linalg.Vector, len(p.basis[0]))
	for i, x := range vec {
		res.Add(p.basis[i].Copy().Scale(x))
	}
	return res
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
