package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"kr/src/dataset"
	"kr/src/utils"
)

type qualityData struct {
	Images []qualityImage `json:"images"`
}

type qualityImage struct {
	Name           string        `json:"name"`
	ReferenceCount int           `json:"referenceCount"`
	ReferenceIssue string        `json:"referenceIssue,omitempty"`
	RANSAC         methodQuality `json:"ransac"`
	SVD            methodQuality `json:"svd"`
}

type methodQuality struct {
	Processed    bool      `json:"processed"`
	PointCount   int       `json:"pointCount"`
	Errors       []float64 `json:"errors"`
	MaximumError *float64  `json:"maximumError"`
	Issue        string    `json:"issue,omitempty"`
}

type savedPoints struct {
	Calibration     utils.Matrix3   `json:"calibration"`
	RANSACClusters  json.RawMessage `json:"ransac_clusters"`
	VanishingPoints json.RawMessage `json:"vanishing_points"`
}

func pointDirection(point utils.HomogeneousPoint, calibration utils.Matrix3) (utils.Vec3, bool) {
	scale := math.Max(math.Abs(point.X), math.Max(math.Abs(point.Y), math.Abs(point.W)))
	if scale == 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return utils.Vec3{}, false
	}
	inverse, ok := calibration.Inverse()
	if !ok {
		return utils.Vec3{}, false
	}
	direction := inverse.MulVec3(utils.Vec3{X: point.X / scale, Y: point.Y / scale, Z: point.W / scale})
	direction.Y = -direction.Y
	return utils.Normalize3(direction)
}

func matchAngles(points, reference []utils.Vec3) []float64 {
	bestErrors := []float64{}
	if len(points) == 0 || len(points) > len(reference) {
		return bestErrors
	}
	bestSum := math.Inf(1)
	used := make([]bool, len(reference))
	errors := make([]float64, len(points))
	var search func(int, float64)
	search = func(index int, sum float64) {
		if index == len(points) {
			if sum < bestSum {
				bestSum = sum
				bestErrors = append([]float64{}, errors...)
			}
			return
		}
		for referenceIndex, target := range reference {
			if used[referenceIndex] {
				continue
			}
			sign := 1.0
			if utils.Dot3(points[index], target) < 0 {
				sign = -1
			}
			difference := utils.Vec3{X: points[index].X - sign*target.X, Y: points[index].Y - sign*target.Y, Z: points[index].Z - sign*target.Z}
			sumVector := utils.Vec3{X: points[index].X + sign*target.X, Y: points[index].Y + sign*target.Y, Z: points[index].Z + sign*target.Z}
			errors[index] = math.Min(90, 2*math.Atan2(math.Sqrt(utils.Dot3(difference, difference)), math.Sqrt(utils.Dot3(sumVector, sumVector)))*180/math.Pi)
			used[referenceIndex] = true
			search(index+1, sum+errors[index])
			used[referenceIndex] = false
		}
	}
	search(0, 0)
	return bestErrors
}

func evaluatePoints(points []utils.HomogeneousPoint, reference []utils.Vec3, calibration utils.Matrix3) methodQuality {
	result := methodQuality{Processed: true, PointCount: len(points), Errors: []float64{}}
	if len(points) > 3 {
		result.Issue = "В отчёте больше трёх точек; угловая оценка недоступна."
		return result
	}
	directions := make([]utils.Vec3, 0, len(points))
	for _, point := range points {
		if direction, ok := pointDirection(point, calibration); ok {
			directions = append(directions, direction)
		}
	}
	if invalidCount := len(points) - len(directions); invalidCount > 0 {
		result.Issue = fmt.Sprintf("Недопустимые точки или калибровка: %d; исключены из угловой оценки.", invalidCount)
	}
	result.Errors = matchAngles(directions, reference)
	if len(reference) == 3 && len(result.Errors) == 3 && len(points) == 3 {
		maximum := math.Max(result.Errors[0], math.Max(result.Errors[1], result.Errors[2]))
		result.MaximumError = &maximum
	}
	return result
}

func loadQuality(datasetRoot string) (qualityData, error) {
	samples, err := dataset.LoadDataset(datasetRoot)
	if err != nil {
		return qualityData{}, err
	}
	quality := qualityData{Images: make([]qualityImage, 0, len(samples))}
	for _, sample := range samples {
		image := qualityImage{
			Name:   sample.Name,
			RANSAC: methodQuality{Errors: []float64{}},
			SVD:    methodQuality{Errors: []float64{}},
		}
		reference, referenceErr := dataset.LoadVanishingPoints(filepath.Join(datasetRoot, sample.Name, sample.Name+"GroundTruthVP_CamParams.mat"))
		if referenceErr != nil {
			image.ReferenceIssue = "Эталон vp отсутствует или повреждён; угловая оценка недоступна."
		} else {
			image.ReferenceCount = len(reference)
		}
		content, readErr := os.ReadFile(filepath.Join(resultsDir, sample.Name, "report.json"))
		var saved savedPoints
		if readErr != nil || json.Unmarshal(content, &saved) != nil {
			image.RANSAC.Issue = "Файл report.json отсутствует или повреждён."
			image.SVD.Issue = image.RANSAC.Issue
		} else {
			if len(saved.RANSACClusters) > 0 {
				var clusters []struct {
					Point utils.HomogeneousPoint `json:"point"`
				}
				if json.Unmarshal(saved.RANSACClusters, &clusters) != nil {
					image.RANSAC.Issue = "В отчёте повреждены точки ransac_clusters."
				} else {
					points := make([]utils.HomogeneousPoint, 0, len(clusters))
					for _, cluster := range clusters {
						points = append(points, cluster.Point)
					}
					image.RANSAC = evaluatePoints(points, reference, saved.Calibration)
				}
			} else {
				image.RANSAC.Issue = "В отчёте отсутствуют точки ransac_clusters."
			}
			if len(saved.VanishingPoints) > 0 {
				var points []utils.HomogeneousPoint
				if json.Unmarshal(saved.VanishingPoints, &points) != nil {
					image.SVD.Issue = "В отчёте повреждены точки vanishing_points."
				} else {
					image.SVD = evaluatePoints(points, reference, saved.Calibration)
				}
			} else {
				image.SVD.Issue = "В отчёте отсутствуют точки vanishing_points."
			}
		}
		quality.Images = append(quality.Images, image)
	}
	return quality, nil
}
