package pcaprune

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"math"

	"github.com/unixpickle/imagecompress/blocker"
	"github.com/unixpickle/num-analysis/linalg"
)

const DefaultBlockSize = 8

// A Compressor uses PCA to compress images by pruning
// the least significant principle components of small
// blocks in an image.
type Compressor struct {
	basisSize int
	blockSize int
}

// NewCompressor is like NewCompressorBlockSize, but
// it uses the DefaultBlockSize.
func NewCompressor(quality float64) *Compressor {
	return NewCompressorBlockSize(quality, DefaultBlockSize)
}

// NewCompressorBlockSize creates a Compressor that
// uses the given quality and block size.
func NewCompressorBlockSize(quality float64, blockSize int) *Compressor {
	basisSize := int(quality * float64(blockSize*blockSize))
	if basisSize < 1 {
		basisSize = 1
	} else if basisSize > blockSize*blockSize {
		basisSize = blockSize * blockSize
	}
	return &Compressor{basisSize: basisSize, blockSize: blockSize}
}

// Compress compresses an image and returns a binary
// encoding of the result.
func (c *Compressor) Compress(i image.Image) []byte {
	var w bytes.Buffer
	binary.Write(&w, encodingEndian, uint32(i.Bounds().Dx()))
	binary.Write(&w, encodingEndian, uint32(i.Bounds().Dy()))

	imageBlocks := blocker.Blocks(i, c.blockSize)
	reducer, err := newPCAReducer(imageBlocks, c.basisSize)
	if err != nil {
		panic(err)
	}

	reducer.WriteTo(&w)

	reducedBlocks := make([]linalg.Vector, len(imageBlocks))
	var maxValue float64
	var minValue float64
	for i, block := range imageBlocks {
		reducedBlocks[i] = reducer.Reduce(block)
		for j, x := range reducedBlocks[i] {
			if j == 0 && i == 0 {
				maxValue = x
				minValue = x
			} else {
				maxValue = math.Max(maxValue, x)
				minValue = math.Min(minValue, x)
			}
		}
	}

	binary.Write(&w, encodingEndian, float64(minValue))
	binary.Write(&w, encodingEndian, float64(maxValue))

	for _, block := range reducedBlocks {
		for _, x := range block {
			val := 255.0 * (x - minValue) / (maxValue - minValue)
			rounded := byte(val + 0.5)
			w.WriteByte(rounded)
		}
	}

	return w.Bytes()
}

// Decompress decodes image data that was encoded
// by Compress.
func (c *Compressor) Decompress(b []byte) (image.Image, error) {
	r := bytes.NewBuffer(b)

	var width, height uint32
	if err := binary.Read(r, encodingEndian, &width); err != nil {
		return nil, errors.New("failed to read width field: " + err.Error())
	}
	if err := binary.Read(r, encodingEndian, &height); err != nil {
		return nil, errors.New("failed to read height field: " + err.Error())
	}

	expander, err := readPCAExpander(r)
	if err != nil {
		return nil, errors.New("failed to read PCA expander: " + err.Error())
	} else if len(expander.basis[0]) != c.blockSize*c.blockSize {
		return nil, errors.New("block size mismatch")
	}

	var minValue, maxValue float64
	if err := binary.Read(r, encodingEndian, &minValue); err != nil {
		return nil, errors.New("failed to read min value: " + err.Error())
	}
	if err := binary.Read(r, encodingEndian, &maxValue); err != nil {
		return nil, errors.New("failed to read max value: " + err.Error())
	}

	rect := image.Rect(0, 0, int(width), int(height))
	blockCount := blocker.Count(rect, c.blockSize)
	imageBlocks := make([]linalg.Vector, blockCount)
	for i := range imageBlocks {
		reducedBlock := make(linalg.Vector, len(expander.basis))
		for j := range reducedBlock {
			if val, err := r.ReadByte(); err != nil {
				return nil, errors.New("failed to read data: " + err.Error())
			} else {
				num := ((float64(val) / 255.0) * (minValue - maxValue)) - minValue
				reducedBlock[j] = num
			}
		}
		imageBlocks[i] = expander.Expand(reducedBlock)
	}

	return blocker.Image(rect.Dx(), rect.Dy(), imageBlocks, c.blockSize), nil
}
