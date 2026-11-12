package algorithm

import (
	"math"
	"math/rand"

	"kr/src/utils"
)

type Config struct {
	AngleThreshold float64
	Confidence     float64
	MaxIterations  int
	Seed           int64
}

type Cluster struct {
	Point    utils.HomogeneousPoint `json:"point"`
	Inliers  []int                  `json:"-"`
	Segments []utils.Segment        `json:"-"`
}

func RequiredRANSACIterations(r, p float64) int {
	if r <= 0 {
		return math.MaxInt
	}
	if r >= 1 {
		return 1
	}
	return int(math.Ceil(math.Log(1-p) / math.Log(1-r*r)))
}

func linePairHypothesis(first, second utils.Segment) (utils.HomogeneousPoint, bool) {
	firstLine, ok := utils.SegmentLine(first)
	if !ok {
		return utils.HomogeneousPoint{}, false
	}
	secondLine, ok := utils.SegmentLine(second)
	if !ok {
		return utils.HomogeneousPoint{}, false
	}
	point, ok := utils.Normalize3(utils.Cross3(
		utils.Vec3{X: firstLine.A, Y: firstLine.B, Z: firstLine.C},
		utils.Vec3{X: secondLine.A, Y: secondLine.B, Z: secondLine.C},
	))
	return utils.HomogeneousPoint{X: point.X, Y: point.Y, W: point.Z}, ok
}

func angleToVanishingPoint(point utils.HomogeneousPoint, segment utils.Segment) float64 {
	segmentX := segment.B.X - segment.A.X
	segmentY := segment.B.Y - segment.A.Y
	segmentLength := math.Hypot(segmentX, segmentY)
	if segmentLength == 0 {
		return math.Inf(1)
	}

	midpointX := (segment.A.X + segment.B.X) / 2
	midpointY := (segment.A.Y + segment.B.Y) / 2
	rayX := point.X - midpointX*point.W
	rayY := point.Y - midpointY*point.W
	rayLength := math.Hypot(rayX, rayY)
	if rayLength == 0 {
		return 0
	}

	cosine := math.Abs(segmentX/segmentLength*(rayX/rayLength) + segmentY/segmentLength*(rayY/rayLength))
	return math.Acos(math.Min(1, cosine))
}

func findLineGroupRANSAC(lines []utils.Segment, config Config) Cluster {
	rng := rand.New(rand.NewSource(config.Seed))
	best := Cluster{}
	limit := config.MaxIterations
	for iteration := 0; iteration < limit; iteration++ {
		first := rng.Intn(len(lines))
		second := rng.Intn(len(lines) - 1)
		if second >= first {
			second++
		}
		point, ok := linePairHypothesis(lines[first], lines[second])
		if !ok {
			continue
		}

		var inliers []int
		for index, segment := range lines {
			if angleToVanishingPoint(point, segment) < config.AngleThreshold {
				inliers = append(inliers, index)
			}
		}
		if len(inliers) > len(best.Inliers) {
			segments := make([]utils.Segment, len(inliers))
			for index, lineIndex := range inliers {
				segments[index] = lines[lineIndex]
			}
			best = Cluster{Point: point, Inliers: inliers, Segments: segments}
			iterations := RequiredRANSACIterations(float64(len(inliers))/float64(len(lines)), config.Confidence)
			if iterations < limit {
				limit = iterations
			}
		}
	}
	return best
}

func GroupLinesRANSAC(lines []utils.Segment, config Config) []Cluster {
	remaining := append([]utils.Segment(nil), lines...)
	var clusters []Cluster
	for len(clusters) < 3 && len(remaining) >= 2 {
		cluster := findLineGroupRANSAC(remaining, config)
		if len(cluster.Inliers) < 2 {
			break
		}
		clusters = append(clusters, cluster)

		assigned := make([]bool, len(remaining))
		for _, index := range cluster.Inliers {
			assigned[index] = true
		}
		unassigned := make([]utils.Segment, 0, len(remaining)-len(cluster.Inliers))
		for index, segment := range remaining {
			if !assigned[index] {
				unassigned = append(unassigned, segment)
			}
		}
		remaining = unassigned
	}
	return clusters
}
