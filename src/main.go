package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"kr/src/algorithm"
	"kr/src/dataset"
	"kr/src/utils"
)

type report struct {
	Calibration          utils.Matrix3            `json:"calibration"`
	RANSACClusters       []algorithm.Cluster      `json:"ransac_clusters"`
	VanishingPoints      []utils.HomogeneousPoint `json:"vanishing_points"`
	OrthogonalDirections []utils.Vec3             `json:"orthogonal_directions"`
}

func main() {
	limit := flag.Int("limit", 0, "maximum images")
	flag.Parse()
	samples, err := dataset.LoadDataset("YorkUrbanDB")
	if err != nil {
		panic(err)
	}
	for i, s := range samples {
		if *limit > 0 && i >= *limit {
			break
		}
		img, err := dataset.LoadGrayJPEG(s.ImagePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		segments := algorithm.DetectLineSegmentsLSD(img)
		config := algorithm.Config{AngleThreshold: .035, Confidence: .99, MaxIterations: 2000, Seed: int64(i + 1)}
		clusters := algorithm.GroupLinesRANSAC(segments, config)
		r := report{
			Calibration:    s.Calibration,
			RANSACClusters: clusters,
		}
		var directions []utils.Vec3
		for _, c := range clusters {
			direction, point, ok := algorithm.RefineVanishingPointSVD(c.Segments, s.Calibration)
			if ok {
				directions = append(directions, direction)
				r.VanishingPoints = append(r.VanishingPoints, point)
			}
		}
		basis, ok := algorithm.BuildOrthogonalBasis(directions)
		if ok {
			r.OrthogonalDirections = append(r.OrthogonalDirections, basis[:]...)
		}
		dir := filepath.Join("output", s.Name)
		os.MkdirAll(dir, 0755)
		outputs := map[string]image.Image{
			"lsd.png":    algorithm.RenderLSDOverlay(img, segments),
			"ransac.png": algorithm.RenderRANSACOverlay(img, clusters),
			"tsp.png":    algorithm.RenderVanishingPointsOverlay(img, r.VanishingPoints, s.Calibration),
		}
		for filename, canvas := range outputs {
			f, err := os.Create(filepath.Join(dir, filename))
			if err != nil {
				panic(err)
			}
			if err := png.Encode(f, canvas); err != nil {
				f.Close()
				panic(err)
			}
			if err := f.Close(); err != nil {
				panic(err)
			}
		}
		b, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "report.json"), b, 0644); err != nil {
			panic(err)
		}
		fmt.Println(s.Name)
	}
}
