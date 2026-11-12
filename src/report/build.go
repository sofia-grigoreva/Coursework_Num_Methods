package main

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
)

//go:embed assets/*
var assets embed.FS

func loadExamples(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	examples := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() && len(name) == 8 && name[0] == 'P' {
			examples = append(examples, name)
		}
	}
	return examples, nil
}

func writeSite(output string, examples []string, quality qualityData) error {
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	for _, name := range []string{"index.html", "app.js", "quality.js", "style.css"} {
		content, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(output, name), content, 0644); err != nil {
			return err
		}
	}
	content, err := json.Marshal(examples)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "data.json"), content, 0644); err != nil {
		return err
	}
	content, err = json.Marshal(quality)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "quality.json"), content, 0644)
}
