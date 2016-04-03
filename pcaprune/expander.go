package pcaprune

import (
	"encoding/binary"
	"io"

	"github.com/unixpickle/num-analysis/linalg"
)

type pcaExpander struct {
	basis []linalg.Vector
}

func readPCAExpander(r io.Reader) (*pcaExpander, error) {
	var count, dimension uint32
	if err := binary.Read(r, encodingEndian, &count); err != nil {
		return nil, err
	}
	if err := binary.Read(r, encodingEndian, &dimension); err != nil {
		return nil, err
	}

	res := &pcaExpander{basis: make([]linalg.Vector, count)}

	for i := 0; i < count; i++ {
		vec := make(linalg.Vector, dimension)
		for j := 0; j < int(dimension); j++ {
			var val float32
			if err := binary.Read(r, encodingEndian, &val); err != nil {
				return nil, err
			}
			vec[j] = float64(val)
		}
		res.basis[i] = vec
	}

	return res, nil
}

func (p *pcaExpander) Expand(vec linalg.Vector) linalg.Vector {
	res := make(linalg.Vector, len(p.basis[0]))
	for i, x := range vec {
		res.Add(p.basis[i].Copy().Scale(x))
	}
	return res
}
