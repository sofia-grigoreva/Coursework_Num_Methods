package dataset

import (
	"fmt"

	"kr/src/utils"
)

// LoadVanishingPoints reads the original York reference vectors, stored as columns.
func LoadVanishingPoints(path string) ([]utils.Vec3, error) {
	matrices, err := readMAT(path)
	if err != nil {
		return nil, err
	}
	matrix, ok := matrices["vp"]
	if !ok || len(matrix.Dims) != 2 || matrix.Dims[0] != 3 || matrix.Dims[1] != 3 || len(matrix.Doubles) != 9 {
		return nil, fmt.Errorf("%s: expected a 3-by-3 vp matrix", path)
	}
	points := make([]utils.Vec3, 0, 3)
	for column := 0; column < 3; column++ {
		offset := 3 * column
		point, valid := utils.Normalize3(utils.Vec3{X: matrix.Doubles[offset], Y: matrix.Doubles[offset+1], Z: matrix.Doubles[offset+2]})
		if !valid {
			return nil, fmt.Errorf("%s: invalid vp column %d", path, column+1)
		}
		points = append(points, point)
	}
	return points, nil
}
