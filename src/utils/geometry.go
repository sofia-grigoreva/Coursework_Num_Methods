package utils

import "math"

const epsilon = 1e-12

type Vec2 struct {
	X float64
	Y float64
}

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Segment struct {
	A Vec2
	B Vec2
}

type HomogeneousLine struct {
	A float64
	B float64
	C float64
}

type HomogeneousPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
}

func Dot3(a, b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func Cross3(a, b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func Normalize3(v Vec3) (Vec3, bool) {
	scale := math.Max(math.Abs(v.X), math.Max(math.Abs(v.Y), math.Abs(v.Z)))
	if scale == 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return Vec3{}, false
	}
	scaled := Vec3{X: v.X / scale, Y: v.Y / scale, Z: v.Z / scale}
	length := math.Sqrt(Dot3(scaled, scaled))
	if scale < epsilon/length {
		return Vec3{}, false
	}
	return Vec3{X: scaled.X / length, Y: scaled.Y / length, Z: scaled.Z / length}, true
}

func SegmentLine(segment Segment) (HomogeneousLine, bool) {
	a := segment.A.Y - segment.B.Y
	b := segment.B.X - segment.A.X
	c := segment.A.X*segment.B.Y - segment.B.X*segment.A.Y
	length := math.Hypot(a, b)
	if length < epsilon {
		return HomogeneousLine{}, false
	}
	return HomogeneousLine{A: a / length, B: b / length, C: c / length}, true
}

type Matrix3 [3][3]float64

func (m Matrix3) MulVec3(v Vec3) Vec3 {
	return Vec3{
		X: m[0][0]*v.X + m[0][1]*v.Y + m[0][2]*v.Z,
		Y: m[1][0]*v.X + m[1][1]*v.Y + m[1][2]*v.Z,
		Z: m[2][0]*v.X + m[2][1]*v.Y + m[2][2]*v.Z,
	}
}

func (m Matrix3) Transpose() Matrix3 {
	return Matrix3{{m[0][0], m[1][0], m[2][0]}, {m[0][1], m[1][1], m[2][1]}, {m[0][2], m[1][2], m[2][2]}}
}

func (m Matrix3) Inverse() (Matrix3, bool) {
	det := m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
	if math.Abs(det) < epsilon {
		return Matrix3{}, false
	}
	invDet := 1 / det
	return Matrix3{
		{(m[1][1]*m[2][2] - m[1][2]*m[2][1]) * invDet, (m[0][2]*m[2][1] - m[0][1]*m[2][2]) * invDet, (m[0][1]*m[1][2] - m[0][2]*m[1][1]) * invDet},
		{(m[1][2]*m[2][0] - m[1][0]*m[2][2]) * invDet, (m[0][0]*m[2][2] - m[0][2]*m[2][0]) * invDet, (m[0][2]*m[1][0] - m[0][0]*m[1][2]) * invDet},
		{(m[1][0]*m[2][1] - m[1][1]*m[2][0]) * invDet, (m[0][1]*m[2][0] - m[0][0]*m[2][1]) * invDet, (m[0][0]*m[1][1] - m[0][1]*m[1][0]) * invDet},
	}, true
}
