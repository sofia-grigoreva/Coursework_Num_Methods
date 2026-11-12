package render

import (
	"image"
	"image/color"
	"math"

	"kr/src/algorithm"
	"kr/src/dataset"
	"kr/src/utils"
)

func DrawSegment(canvas *image.RGBA, segment utils.Segment, lineColor color.RGBA) {
	x0, y0 := int(math.Round(segment.A.X)), int(math.Round(segment.A.Y))
	x1, y1 := int(math.Round(segment.B.X)), int(math.Round(segment.B.Y))
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		canvas.SetRGBA(x0, y0, lineColor)
		if x0 == x1 && y0 == y1 {
			return
		}
		twice := 2 * err
		if twice >= dy {
			err += dy
			x0 += sx
		}
		if twice <= dx {
			err += dx
			y0 += sy
		}
	}
}

func RenderRANSACOverlay(source dataset.GrayImage, clusters []algorithm.Cluster) *image.RGBA {
	canvas := RenderLSDOverlay(source, nil)
	colors := []color.RGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}}
	for group, cluster := range clusters {
		for _, segment := range cluster.Segments {
			DrawSegment(canvas, segment, colors[group%len(colors)])
		}
	}
	return canvas
}

func RenderVanishingPointsOverlay(source dataset.GrayImage, points []utils.HomogeneousPoint, k utils.Matrix3) *image.RGBA {
	canvas := RenderLSDOverlay(source, nil)
	colors := []color.RGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}}
	for index, point := range points {
		DrawCircle(canvas, vanishingPointForDisplay(point, k, canvas.Bounds()), colors[index%len(colors)])
	}
	return canvas
}

func DrawCircle(canvas *image.RGBA, point utils.Vec2, marker color.RGBA) {
	x, y := int(math.Round(point.X)), int(math.Round(point.Y))
	const radius = 6
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				canvas.SetRGBA(x+dx, y+dy, marker)
			}
		}
	}
}

func RenderLSDOverlay(source dataset.GrayImage, segments []utils.Segment) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, source.Width, source.Height))
	for y := 0; y < source.Height; y++ {
		for x := 0; x < source.Width; x++ {
			value := source.At(x, y)
			canvas.SetRGBA(x, y, color.RGBA{R: value, G: value, B: value, A: 255})
		}
	}
	for _, segment := range segments {
		DrawSegment(canvas, segment, color.RGBA{R: 255, G: 220, A: 255})
	}
	return canvas
}

func vanishingPointForDisplay(point utils.HomogeneousPoint, k utils.Matrix3, bounds image.Rectangle) utils.Vec2 {
	cx, cy := k[0][2], k[1][2]
	if point.W != 0 {
		x, y := point.X, point.Y
		if x >= float64(bounds.Min.X) && x < float64(bounds.Max.X) && y >= float64(bounds.Min.Y) && y < float64(bounds.Max.Y) {
			return utils.Vec2{X: x, Y: y}
		}
		return rayToBorder(cx, cy, x, y, bounds)
	}
	return rayToBorder(cx, cy, cx+point.X, cy+point.Y, bounds)
}

func rayToBorder(cx, cy, x, y float64, bounds image.Rectangle) utils.Vec2 {
	dx, dy := x-cx, y-cy
	if dx == 0 && dy == 0 {
		return utils.Vec2{X: cx, Y: cy}
	}
	t := math.Inf(1)
	if dx > 0 {
		t = math.Min(t, (float64(bounds.Max.X-1)-cx)/dx)
	} else if dx < 0 {
		t = math.Min(t, (float64(bounds.Min.X)-cx)/dx)
	}
	if dy > 0 {
		t = math.Min(t, (float64(bounds.Max.Y-1)-cy)/dy)
	} else if dy < 0 {
		t = math.Min(t, (float64(bounds.Min.Y)-cy)/dy)
	}
	return utils.Vec2{X: cx + t*dx, Y: cy + t*dy}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
