package dataset

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"io"
	"math"
	"os"
	"path/filepath"

	"kr/src/utils"
)

type GrayImage struct {
	Width, Height int
	Pixels        []uint8
}

func (image GrayImage) At(x, y int) uint8 {
	return image.Pixels[y*image.Width+x]
}

type Sample struct {
	Name        string
	ImagePath   string
	Calibration utils.Matrix3
}

func LoadGrayJPEG(path string) (GrayImage, error) {
	file, err := os.Open(path)
	if err != nil {
		return GrayImage{}, err
	}
	defer file.Close()
	source, _, err := image.Decode(file)
	if err != nil {
		return GrayImage{}, err
	}
	bounds := source.Bounds()
	gray := GrayImage{Width: bounds.Dx(), Height: bounds.Dy(), Pixels: make([]uint8, bounds.Dx()*bounds.Dy())}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := source.At(x, y).RGBA()
			gray.Pixels[(y-bounds.Min.Y)*gray.Width+x-bounds.Min.X] = color.GrayModel.Convert(color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: 0xffff}).(color.Gray).Y
		}
	}
	return gray, nil
}

func LoadDataset(root string) ([]Sample, error) {
	calibration, err := loadCalibration(filepath.Join(root, "cameraParameters.mat"))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var samples []Sample
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() && len(name) == 8 && name[0] == 'P' {
			samples = append(samples, Sample{
				Name:        name,
				ImagePath:   filepath.Join(root, name, name+".jpg"),
				Calibration: calibration,
			})
		}
	}
	return samples, nil
}

func loadCalibration(path string) (utils.Matrix3, error) {
	matrices, err := readMAT(path)
	if err != nil {
		return utils.Matrix3{}, err
	}
	focal, okA := matrices["focal"]
	pixelSize, okB := matrices["pixelSize"]
	principal, okC := matrices["pp"]
	if !okA || !okB || !okC || len(focal.Doubles) != 1 || len(pixelSize.Doubles) != 1 || len(principal.Doubles) != 2 {
		return utils.Matrix3{}, fmt.Errorf("invalid camera calibration")
	}
	f := focal.Doubles[0] / pixelSize.Doubles[0]
	return utils.Matrix3{{f, 0, principal.Doubles[0] - 1}, {0, f, principal.Doubles[1] - 1}, {0, 0, 1}}, nil
}

const (
	miDouble     = 9
	miMatrix     = 14
	miCompressed = 15
)

type matMatrix struct {
	Name    string
	Dims    []int
	Doubles []float64
}

type matElement struct {
	Type int
	Data []byte
}

func readMAT(path string) (map[string]matMatrix, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 128 || string(data[126:128]) != "IM" {
		return nil, fmt.Errorf("%s: not a little-endian MATLAB v5 file", path)
	}
	matrices := make(map[string]matMatrix)
	for offset := 128; offset < len(data); {
		element, next, err := readElement(data, offset)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		offset = next
		if element.Type != miMatrix {
			return nil, fmt.Errorf("%s: unsupported MATLAB element type %d; expected an uncompressed matrix", path, element.Type)
		}
		matrix, err := parseMatrix(element.Data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		matrices[matrix.Name] = matrix
	}
	return matrices, nil
}

func readElement(data []byte, offset int) (matElement, int, error) {
	if offset+8 > len(data) {
		return matElement{}, offset, io.ErrUnexpectedEOF
	}
	first := binary.LittleEndian.Uint32(data[offset:])
	if first == miCompressed {
		return matElement{}, offset, fmt.Errorf("unsupported compressed MATLAB element")
	}
	second := binary.LittleEndian.Uint32(data[offset+4:])
	if first>>16 != 0 {
		size := int(first >> 16)
		if size > 4 {
			return matElement{}, offset, fmt.Errorf("invalid small element size %d", size)
		}
		return matElement{Type: int(first & 0xffff), Data: data[offset+4 : offset+4+size]}, offset + 8, nil
	}
	size := int(second)
	start := offset + 8
	end := start + size
	if end > len(data) {
		return matElement{}, offset, io.ErrUnexpectedEOF
	}
	next := end + (8-size%8)%8
	if next > len(data) {
		return matElement{}, offset, io.ErrUnexpectedEOF
	}
	return matElement{Type: int(first), Data: data[start:end]}, next, nil
}

func parseMatrix(data []byte) (matMatrix, error) {
	flags, offset, err := readElement(data, 0)
	if err != nil || len(flags.Data) < 4 {
		return matMatrix{}, fmt.Errorf("invalid matrix flags")
	}
	if binary.LittleEndian.Uint32(flags.Data)&0xff != 6 {
		return matMatrix{}, fmt.Errorf("unsupported matrix class; expected double")
	}
	dimensions, offset, err := readElement(data, offset)
	if err != nil || len(dimensions.Data)%4 != 0 {
		return matMatrix{}, fmt.Errorf("invalid matrix dimensions")
	}
	dims := make([]int, len(dimensions.Data)/4)
	for i := range dims {
		dims[i] = int(int32(binary.LittleEndian.Uint32(dimensions.Data[i*4:])))
	}
	name, offset, err := readElement(data, offset)
	if err != nil {
		return matMatrix{}, err
	}
	realPart, _, err := readElement(data, offset)
	if err != nil {
		return matMatrix{}, err
	}
	if realPart.Type != miDouble {
		return matMatrix{}, fmt.Errorf("matrix %q: unsupported numeric type %d; expected double", name.Data, realPart.Type)
	}
	if len(realPart.Data)%8 != 0 {
		return matMatrix{}, fmt.Errorf("matrix %q: invalid double data size", name.Data)
	}
	matrix := matMatrix{Name: string(name.Data), Dims: dims, Doubles: make([]float64, len(realPart.Data)/8)}
	for i := range matrix.Doubles {
		matrix.Doubles[i] = math.Float64frombits(binary.LittleEndian.Uint64(realPart.Data[i*8:]))
	}
	count := 1
	for _, dimension := range dims {
		if dimension < 0 || (dimension > 0 && count > len(matrix.Doubles)/dimension) {
			return matMatrix{}, fmt.Errorf("matrix %q: dimensions exceed stored values", matrix.Name)
		}
		count *= dimension
	}
	if count != len(matrix.Doubles) {
		return matMatrix{}, fmt.Errorf("matrix %q: dimensions differ from stored values", matrix.Name)
	}
	return matrix, nil
}
