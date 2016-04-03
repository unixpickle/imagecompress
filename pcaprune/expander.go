package pcaprune

import "github.com/unixpickle/num-analysis/linalg"

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
