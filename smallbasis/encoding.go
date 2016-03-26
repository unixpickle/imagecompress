package smallbasis

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	basisHeadingSparse = 0
	basisHeadingDense  = 1
)

var encodedByteOrder = binary.LittleEndian

type compressedImage struct {
	// UsedBasis contains the indices of the basis
	// vectors that are used in this compressedImage.
	// This list should be sorted in ascending order.
	UsedBasis []int

	// Blocks contains an array of blocks, encoded as
	// linear combinations of the used basis vectors.
	Blocks [][]float64

	BlockSize int
	Width     int
	Height    int
}

// decodeCompressedImage unpacks a binary representation
// of a compressedImage.
// You must know the block size ahead of time to decode the
// file, but this is reasonable since you must also know the
// basis ahead of time to actually utilize the compressedImage.
func decodeCompressedImage(data []byte, blockSize int) (*compressedImage, error) {
	buf := bytes.NewBuffer(data)

	res := &compressedImage{
		BlockSize: blockSize,
	}

	var width, height uint32
	if err := binary.Read(buf, encodedByteOrder, &width); err != nil {
		return nil, errors.New("missing width field")
	}
	if err := binary.Read(buf, encodedByteOrder, &height); err != nil {
		return nil, errors.New("missing height field")
	}

	res.Width = int(width)
	res.Height = int(height)

	if b, err := buf.ReadByte(); err != nil {
		return nil, errors.New("missing basis heading")
	} else if b == basisHeadingSparse {
		if err := res.decodeSparseBasis(buf); err != nil {
			return nil, err
		}
	} else if b == basisHeadingDense {
		if err := res.decodeDenseBasis(buf); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown basis heading type: 0x%x", b)
	}

	var maxCoeff float64
	if err := binary.Read(buf, encodedByteOrder, &maxCoeff); err != nil {
		return nil, errors.New("missing maximum coefficient value")
	}

	horizBlockCount := res.Width / blockSize
	vertBlockCount := res.Height / blockSize
	if res.Width%blockSize != 0 {
		horizBlockCount++
	}
	if res.Height%blockSize != 0 {
		vertBlockCount++
	}
	for i := 0; i < horizBlockCount*vertBlockCount*3; i++ {
		if err := res.decodeNextBlock(maxCoeff, buf); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// Encode generates a binary representation of this image.
func (i *compressedImage) Encode() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, encodedByteOrder, uint32(i.Width))
	binary.Write(&buf, encodedByteOrder, uint32(i.Height))

	fullBasisSize := i.BlockSize * i.BlockSize
	sparseBasisSize := len(i.UsedBasis) * 32
	if sparseBasisSize < fullBasisSize {
		buf.WriteByte(basisHeadingSparse)
		buf.Write(i.encodeSparseBasis())
	} else {
		buf.WriteByte(basisHeadingDense)
		buf.Write(i.encodeDenseBasis())
	}

	maxCoeff := i.maxCoefficient()
	binary.Write(&buf, encodedByteOrder, maxCoeff)

	for _, block := range i.Blocks {
		for _, blockValue := range block {
			blockValue += maxCoeff
			blockValue /= maxCoeff * 2
			blockValue *= 0xff
			num := roundFloat(blockValue)
			buf.WriteByte(byte(num))
		}
	}

	return buf.Bytes()
}

// encodeSparseBasis generates a list of basis element
// indices, each encoded as 32-bits.
//
// This is good when we are using a small fraction of the
// original basis, since we only need space for the vectors
// we are using.
func (i *compressedImage) encodeSparseBasis() []byte {
	res := make([]byte, (len(i.UsedBasis)+1)*4)
	encodedByteOrder.PutUint32(res, uint32(len(i.UsedBasis)))
	for i, vec := range i.UsedBasis {
		encodedByteOrder.PutUint32(res[(i+1)*4:], uint32(vec))
	}
	return res
}

// encodeDenseBasis generates a bitmap indicating which
// basis elements to use.
//
// This is good when we are using a relatively large
// fraction of the original bases, since each used and
// unused basis element only requires one bit of space.
func (i *compressedImage) encodeDenseBasis() []byte {
	bitCount := i.BlockSize * i.BlockSize
	byteCount := bitCount / 8
	if bitCount%8 != 0 {
		byteCount++
	}

	res := make([]byte, byteCount)
	for _, vec := range i.UsedBasis {
		byteIndex := vec >> 3
		bitIndex := uint(vec & 7)
		res[byteIndex] |= (1 << bitIndex)
	}
	return res
}

// maxCoefficient gets the basis coefficient with the
// biggest magnitude in any block of the image.
func (i *compressedImage) maxCoefficient() float64 {
	var coeff float64
	for _, block := range i.Blocks {
		for _, c := range block {
			coeff = math.Max(coeff, math.Abs(c))
		}
	}
	return coeff
}

// decodeSparseBasis performs the inverse of
// encodeSparseBasis.
func (i *compressedImage) decodeSparseBasis(r *bytes.Buffer) error {
	var count uint32
	if err := binary.Read(r, encodedByteOrder, &count); err != nil {
		return errors.New("missing sparse vector count")
	}

	i.UsedBasis = make([]int, int(count))
	for index := range i.UsedBasis {
		var vec uint32
		if err := binary.Read(r, encodedByteOrder, &vec); err != nil {
			return errors.New("could not read basis vector")
		}
		i.UsedBasis[index] = int(vec)
	}

	return nil
}

// decodeDenseBasis performs the inverse of
// encodeDenseBasis.
func (i *compressedImage) decodeDenseBasis(r *bytes.Buffer) error {
	bitCount := i.BlockSize * i.BlockSize
	byteCount := bitCount / 8
	if bitCount%8 != 0 {
		byteCount++
	}

	bytes := make([]byte, byteCount)
	if n, err := r.Read(bytes); err != nil || n < len(bytes) {
		return errors.New("could not read basis bitmap")
	}

	i.UsedBasis = []int{}

	byteIndex := 0
	bitIndex := uint(0)
	for index := 0; index < bitCount; index++ {
		bit := (bytes[byteIndex] & (1 << bitIndex)) != 0
		if bit {
			i.UsedBasis = append(i.UsedBasis, index)
		}
		if bitIndex == 7 {
			bitIndex = 0
			byteIndex++
		} else {
			bitIndex++
		}
	}

	return nil
}

// decodeNextBlock reads a block (i.e. a linear
// combination of basis vectors) from the buffer.
func (i *compressedImage) decodeNextBlock(maxCoeff float64, r *bytes.Buffer) error {
	block := make([]float64, len(i.UsedBasis))
	for k := 0; k < len(i.UsedBasis); k++ {
		if b, err := r.ReadByte(); err != nil {
			return errors.New("could not read coefficient data")
		} else {
			val := float64(uint8(b))
			val /= 0xff
			val *= maxCoeff * 2
			val -= maxCoeff
			block[k] = val
		}
	}
	i.Blocks = append(i.Blocks, block)
	return nil
}
