package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/unixpickle/imagecompress/smallbasis"
)

type Compressor interface {
	Compress(i image.Image) []byte
	Decompress(d []byte) (image.Image, error)
}

type CompressorGen func(quality float64) Compressor

var Compressors = map[string]CompressorGen{
	"smallbasis": func(q float64) Compressor {
		return smallbasis.NewCompressor(q)
	},
	"ortho16": func(q float64) Compressor {
		return smallbasis.NewCompressorBasis(q, 16, smallbasis.OrthoBasis(16*16))
	},
}

func main() {
	if len(os.Args) != 5 && len(os.Args) != 6 {
		dieUsage()
	}

	compName := os.Args[2]
	gen := Compressors[compName]
	if gen == nil {
		fmt.Fprintln(os.Stderr, "unknown compressor: ", compName)
		os.Exit(1)
	}

	if os.Args[1] == "compress" {
		if len(os.Args) != 6 {
			dieUsage()
		}
		quality, err := strconv.ParseFloat(os.Args[3], 64)
		if err != nil || quality < 0 || quality > 1 {
			fmt.Fprintln(os.Stderr, "invalid quality: ", os.Args[3])
			os.Exit(1)
		}
		if err := compress(gen(quality), os.Args[4], os.Args[5]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if os.Args[1] == "decompress" {
		if len(os.Args) != 5 {
			dieUsage()
		}
		if err := decompress(gen(0), os.Args[3], os.Args[4]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		dieUsage()
	}
}

func compress(c Compressor, inFile, outFile string) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	data := c.Compress(img)
	return ioutil.WriteFile(outFile, data, 0755)
}

func decompress(c Compressor, inFile, outFile string) error {
	data, err := ioutil.ReadFile(inFile)
	if err != nil {
		return err
	}
	img, err := c.Decompress(data)
	if err != nil {
		return err
	}

	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func dieUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <compress> <compressor> <quality> <in.png> <out>\n"+
		"       %s <decompress> <compressor> <in> <out.png>\n\n"+
		"Compressors:\n"+
		" smallbasis       algebraic basis pruning\n"+
		" ortho16          prune a recursive orthogonal basis\n",
		os.Args[0], os.Args[0])
	os.Exit(1)
}
