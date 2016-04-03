package blocker

import (
	"image"
	"image/color"
	"math"

	"github.com/unixpickle/num-analysis/linalg"
)

// Blocks returns a bunch of vectors representing
// square blocks of red, green, and blue pixels
// in the given image.
//
// The blockSize argument specifies the side-length of
// each block of pixels.
//
// If a square block extends past the bounds of the
// image, the overflowing pixel values will be 0's.
func Blocks(i image.Image, blockSize int) []linalg.Vector {
	numRows, numCols := blockCounts(i.Bounds(), blockSize)

	res := make([]linalg.Vector, 0, 3*numRows*numCols)
	for row := 0; row < numRows; row++ {
		for col := 0; col < numCols; col++ {
			startX := i.Bounds().Min.X + col*blockSize
			startY := i.Bounds().Min.Y + row*blockSize
			blocks := make([]linalg.Vector, 3)
			for i := range blocks {
				blocks[i] = make(linalg.Vector, blockSize*blockSize)
			}
			for y := 0; y < blockSize; y++ {
				if y+startY >= i.Bounds().Max.Y {
					continue
				}
				for x := 0; x < blockSize; x++ {
					if x+startX >= i.Bounds().Max.X {
						continue
					}
					px := i.At(x+startX, y+startY)
					r, g, b, _ := px.RGBA()
					idx := y * blockSize
					if y%2 == 0 {
						idx += x
					} else {
						idx += blockSize - (x + 1)
					}
					blocks[0][idx] = float64(r) / 0xffff
					blocks[1][idx] = float64(g) / 0xffff
					blocks[2][idx] = float64(b) / 0xffff
				}
			}
			res = append(res, blocks...)
		}
	}

	return res
}

// Image performs the inverse of Blocks.
func Image(w, h int, blocks []linalg.Vector, blockSize int) image.Image {
	res := image.NewRGBA(image.Rect(0, 0, w, h))
	rows, cols := blockCounts(res.Bounds(), blockSize)

	blockIdx := 0
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			colorBlocks := blocks[blockIdx : blockIdx+3]
			blockIdx += 3
			for y := 0; y < blockSize; y++ {
				if y+row*blockSize >= h {
					continue
				}
				for x := 0; x < blockSize; x++ {
					if x+col*blockSize >= w {
						continue
					}
					pxIdx := y * blockSize
					if y%2 == 0 {
						pxIdx += x
					} else {
						pxIdx += blockSize - (x + 1)
					}
					rVal := math.Min(math.Max(colorBlocks[0][pxIdx], 0), 1)
					gVal := math.Min(math.Max(colorBlocks[1][pxIdx], 0), 1)
					bVal := math.Min(math.Max(colorBlocks[2][pxIdx], 0), 1)
					px := color.RGBA{
						R: uint8(rVal * 0xff),
						G: uint8(gVal * 0xff),
						B: uint8(bVal * 0xff),
						A: 0xff,
					}
					res.Set(x+col*blockSize, y+row*blockSize, px)
				}
			}
		}
	}

	return res
}

// Count returns the number of blocks needed to
// encode an image of the given dimensions.
func Count(b image.Rectangle, blockSize int) int {
	rows, cols := blockCounts(b, blockSize)
	return rows * cols * 3
}

func blockCounts(bounds image.Rectangle, blockSize int) (rows, cols int) {
	cols = bounds.Dx() / blockSize
	if bounds.Dx()%blockSize != 0 {
		cols++
	}

	rows = bounds.Dy() / blockSize
	if bounds.Dy()%blockSize != 0 {
		rows++
	}

	return
}
