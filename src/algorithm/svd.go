package algorithm

import (
	"math"

	"kr/src/utils"
)

func RefineVanishingPointSVD(lines []utils.Segment, k utils.Matrix3) (utils.Vec3, utils.HomogeneousPoint, bool) {
	if _, ok := k.Inverse(); !ok {
		return utils.Vec3{}, utils.HomogeneousPoint{}, false
	}
	// lᵀv = 0 and v = Kd imply (Kᵀl)ᵀd = 0.
	// Section 3.3 of the paper prints K⁻¹l, apparently a typo for line coefficients.
	transposed := k.Transpose()
	rows := make([][3]float64, 0, len(lines))
	for _, s := range lines {
		l, ok := utils.SegmentLine(s)
		if !ok {
			continue
		}
		normal, ok := utils.Normalize3(transposed.MulVec3(utils.Vec3{X: l.A, Y: l.B, Z: l.C}))
		if !ok {
			continue
		}
		rows = append(rows, [3]float64{normal.X, normal.Y, normal.Z})
	}
	if len(rows) < 2 {
		return utils.Vec3{}, utils.HomogeneousPoint{}, false
	}
	_, singular, right, ok := SingularValueDecomposition(rows)
	if !ok || singular[0] == 0 || singular[1] <= epsilon*singular[0] {
		return utils.Vec3{}, utils.HomogeneousPoint{}, false
	}
	direction := utils.Vec3{X: right[0][2], Y: right[1][2], Z: right[2][2]}
	projected := k.MulVec3(direction)
	point := utils.HomogeneousPoint{X: projected.X, Y: projected.Y}
	if math.Abs(projected.Z) > epsilon {
		point.X /= projected.Z
		point.Y /= projected.Z
		point.W = 1
	}
	return direction, point, true
}

const (
	epsilon                       = 1e-12
	orthogonalityToleranceDegrees = 5.0
)

func OrthogonalizeDirections(directions [3]utils.Vec3) ([3]utils.Vec3, bool) {
	rows := make([][3]float64, 3)
	for column, direction := range directions {
		rows[0][column] = direction.X
		rows[1][column] = direction.Y
		rows[2][column] = direction.Z
	}
	left, _, right, ok := SingularValueDecomposition(rows)
	if !ok {
		return [3]utils.Vec3{}, false
	}
	// D = UΣVᵀ, so the nearest orthogonal matrix is U Vᵀ.
	var orthogonal utils.Matrix3
	for row := 0; row < 3; row++ {
		for column := 0; column < 3; column++ {
			for axis := 0; axis < 3; axis++ {
				orthogonal[row][column] += left[row][axis] * right[column][axis]
			}
		}
	}
	return [3]utils.Vec3{
		{X: orthogonal[0][0], Y: orthogonal[1][0], Z: orthogonal[2][0]},
		{X: orthogonal[0][1], Y: orthogonal[1][1], Z: orthogonal[2][1]},
		{X: orthogonal[0][2], Y: orthogonal[1][2], Z: orthogonal[2][2]},
	}, true
}

func BuildOrthogonalBasis(directions []utils.Vec3) ([3]utils.Vec3, bool) {
	if len(directions) < 2 || len(directions) > 3 {
		return [3]utils.Vec3{}, false
	}
	var normalized [3]utils.Vec3
	for i, direction := range directions {
		var ok bool
		normalized[i], ok = utils.Normalize3(direction)
		if !ok {
			return [3]utils.Vec3{}, false
		}
	}
	if len(directions) == 2 {
		if !approximatelyOrthogonal(normalized[0], normalized[1]) {
			return [3]utils.Vec3{}, false
		}
		third, ok := utils.Normalize3(utils.Cross3(normalized[0], normalized[1]))
		if !ok {
			return [3]utils.Vec3{}, false
		}
		normalized[2] = third
		return OrthogonalizeDirections(normalized)
	}
	orthogonalPairs, missingAxis := 0, 0
	for _, pair := range [][2]int{{0, 1}, {0, 2}, {1, 2}} {
		if approximatelyOrthogonal(normalized[pair[0]], normalized[pair[1]]) {
			orthogonalPairs++
			missingAxis = 3 - pair[0] - pair[1]
		}
	}
	// Exactly one orthogonal pair identifies the two supported directions in §4.1.
	if orthogonalPairs == 1 {
		third, ok := utils.Normalize3(utils.Cross3(normalized[(missingAxis+1)%3], normalized[(missingAxis+2)%3]))
		if !ok {
			return [3]utils.Vec3{}, false
		}
		normalized[missingAxis] = third
	}
	return OrthogonalizeDirections(normalized)
}

func approximatelyOrthogonal(a, b utils.Vec3) bool {
	return math.Abs(utils.Dot3(a, b)) <= math.Sin(orthogonalityToleranceDegrees*math.Pi/180)+epsilon
}

// SingularValueDecomposition computes A = UΣVᵀ for a matrix with three columns.
// One-sided Jacobi rotations orthogonalize A's columns directly. Zero rows pad
// two-line systems to three rows. Singular values are returned in descending order.
func SingularValueDecomposition(rows [][3]float64) ([][3]float64, [3]float64, utils.Matrix3, bool) {
	columns := make([][3]float64, max(3, len(rows)))
	scale := 0.0
	for _, row := range rows {
		for _, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, [3]float64{}, utils.Matrix3{}, false
			}
			scale = math.Max(scale, math.Abs(value))
		}
	}
	if scale == 0 {
		scale = 1
	}
	matrixNormSquared := 0.0
	for i, row := range rows {
		for j, value := range row {
			columns[i][j] = value / scale
			matrixNormSquared += columns[i][j] * columns[i][j]
		}
	}
	rankTolerance := 1e-14 * math.Sqrt(matrixNormSquared)
	right := utils.Matrix3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	converged := false
	for sweep := 0; sweep < 100; sweep++ {
		rotated := false
		for _, pair := range [][2]int{{0, 1}, {0, 2}, {1, 2}} {
			p, q := pair[0], pair[1]
			var alpha, beta, gamma float64
			for _, row := range columns {
				alpha += row[p] * row[p]
				beta += row[q] * row[q]
				gamma += row[p] * row[q]
			}
			if math.Min(alpha, beta) <= rankTolerance*rankTolerance ||
				math.Abs(gamma) <= 1e-14*math.Sqrt(alpha)*math.Sqrt(beta) {
				continue
			}
			tau := (beta - alpha) / (2 * gamma)
			tangent := math.Copysign(1, tau) / (math.Abs(tau) + math.Hypot(1, tau))
			cosine := 1 / math.Hypot(1, tangent)
			sine := cosine * tangent
			for i := range columns {
				a, b := columns[i][p], columns[i][q]
				columns[i][p], columns[i][q] = cosine*a-sine*b, sine*a+cosine*b
			}
			for i := 0; i < 3; i++ {
				a, b := right[i][p], right[i][q]
				right[i][p], right[i][q] = cosine*a-sine*b, sine*a+cosine*b
			}
			rotated = true
		}
		if !rotated {
			converged = true
			break
		}
	}
	if !converged {
		return nil, [3]float64{}, utils.Matrix3{}, false
	}
	var singular [3]float64
	for _, row := range columns {
		for j, value := range row {
			singular[j] = math.Hypot(singular[j], value)
		}
	}
	for p := 0; p < 2; p++ {
		for q := p + 1; q < 3; q++ {
			if singular[q] > singular[p] {
				singular[p], singular[q] = singular[q], singular[p]
				for i := range columns {
					columns[i][p], columns[i][q] = columns[i][q], columns[i][p]
				}
				for i := 0; i < 3; i++ {
					right[i][p], right[i][q] = right[i][q], right[i][p]
				}
			}
		}
	}
	left := make([][3]float64, len(columns))
	for column := 0; column < 3; column++ {
		if singular[column] > rankTolerance {
			for row := range left {
				left[row][column] = columns[row][column] / singular[column]
			}
			continue
		}
		// A zero singular value leaves U's column undetermined. Complete its
		// orthonormal basis so U Vᵀ is also defined for rank-deficient D.
		singular[column] = 0
		for candidate := range left {
			for row := range left {
				left[row][column] = 0
			}
			left[candidate][column] = 1
			for pass := 0; pass < 2; pass++ {
				for previous := 0; previous < column; previous++ {
					dot := 0.0
					for row := range left {
						dot += left[row][column] * left[row][previous]
					}
					for row := range left {
						left[row][column] -= dot * left[row][previous]
					}
				}
			}
			norm := 0.0
			for row := range left {
				norm = math.Hypot(norm, left[row][column])
			}
			if norm > 0.5 {
				for row := range left {
					left[row][column] /= norm
				}
				break
			}
		}
	}
	for i := range singular {
		singular[i] *= scale
	}
	return left, singular, right, true
}
