package algorithm

import (
	"math"
	"sort"

	"kr/src/dataset"
	"kr/src/utils"
)

const gradientThreshold = 25.0
const angleTolerance = math.Pi / 8

type pixel struct {
	x, y int
}

func ComputeGradients(img dataset.GrayImage) ([]float64, []float64) {
	n := img.Width * img.Height
	magnitude, angle := make([]float64, n), make([]float64, n)
	for y := 1; y < img.Height-1; y++ {
		for x := 1; x < img.Width-1; x++ {
			gx := float64(img.At(x+1, y)) - float64(img.At(x-1, y))
			gy := float64(img.At(x, y+1)) - float64(img.At(x, y-1))
			i := y*img.Width + x
			magnitude[i] = math.Hypot(gx, gy)
			angle[i] = math.Atan2(gy, gx)
		}
	}
	return magnitude, angle
}

func DetectLineSegmentsLSD(img dataset.GrayImage) []utils.Segment {
	mag, angles := ComputeGradients(img)
	order := make([]int, 0, len(mag))
	for i, v := range mag {
		if v >= gradientThreshold {
			order = append(order, i)
		}
	}
	sort.Slice(order, func(i, j int) bool { return mag[order[i]] > mag[order[j]] })
	used := make([]bool, len(mag))
	logTests := math.Log10(float64(img.Width * img.Height))
	logProbability := math.Log10(angleTolerance / math.Pi)
	var out []utils.Segment
	for _, seed := range order {
		if used[seed] {
			continue
		}
		region := grow(seed, img.Width, img.Height, angles, mag, used)
		if len(region) < 8 {
			continue
		}
		// All region pixels are aligned, so the NFA tail reduces to p^n, p = 1/8.
		if segment, ok := fit(region); ok && logTests+float64(len(region))*logProbability < 0 {
			out = append(out, segment)
		}
	}
	return out
}

func grow(seed, w, h int, angles, mag []float64, used []bool) []pixel {
	queue := []int{seed}
	used[seed] = true
	base := angles[seed]
	var region []pixel
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%w, i/w
		region = append(region, pixel{x, y})
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				ni := ny*w + nx
				if nx < 1 || ny < 1 || nx >= w-1 || ny >= h-1 || used[ni] || mag[ni] < gradientThreshold || angleDiff(angles[ni], base) > angleTolerance {
					continue
				}
				used[ni] = true
				queue = append(queue, ni)
			}
		}
	}
	return region
}

func angleDiff(a, b float64) float64 {
	d := math.Abs(a - b)
	for d > math.Pi {
		d = math.Abs(d - 2*math.Pi)
	}
	if d > math.Pi/2 {
		d = math.Pi - d
	}
	return d
}

func fit(points []pixel) (utils.Segment, bool) {
	var mx, my float64
	for _, p := range points {
		mx += float64(p.x)
		my += float64(p.y)
	}
	mx /= float64(len(points))
	my /= float64(len(points))
	var xx, xy, yy float64
	for _, p := range points {
		dx, dy := float64(p.x)-mx, float64(p.y)-my
		xx += dx * dx
		xy += dx * dy
		yy += dy * dy
	}
	theta := 0.5 * math.Atan2(2*xy, xx-yy)
	dx, dy := math.Cos(theta), math.Sin(theta)
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range points {
		t := (float64(p.x)-mx)*dx + (float64(p.y)-my)*dy
		if t < lo {
			lo = t
		}
		if t > hi {
			hi = t
		}
	}
	if hi-lo < 6 {
		return utils.Segment{}, false
	}
	return utils.Segment{A: utils.Vec2{X: mx + lo*dx, Y: my + lo*dy}, B: utils.Vec2{X: mx + hi*dx, Y: my + hi*dy}}, true
}
