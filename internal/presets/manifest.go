package presets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Goal        string   `json:"goal"`
	Adapter     string   `json:"adapter"`
	Category    string   `json:"category"`
	Persona     string   `json:"persona"`
	Version     string   `json:"version"`
	Files       []string `json:"files"`
	SafetyNotes []string `json:"safety_notes"`
	Author      Author   `json:"author"`
	Path        string   `json:"-"`
}

type Author struct {
	Name   string `json:"name"`
	Handle string `json:"handle"`
}

func List(root string) ([]Manifest, error) {
	entries := []Manifest{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "manifest.yaml" {
			return nil
		}
		manifest, err := LoadManifest(path)
		if err != nil {
			return err
		}
		entries = append(entries, manifest)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Category == entries[j].Category {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Category < entries[j].Category
	})
	return entries, nil
}

func LoadByID(root string, id string) (Manifest, error) {
	manifests, err := List(root)
	if err != nil {
		return Manifest{}, err
	}
	for _, manifest := range manifests {
		if manifest.ID == id {
			return manifest, nil
		}
	}
	return Manifest{}, fmt.Errorf("preset %q not found", id)
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("manifest %s must be JSON-compatible YAML: %w", path, err)
	}
	manifest.Path = path
	if manifest.ID == "" || manifest.Name == "" || manifest.Adapter == "" {
		return Manifest{}, fmt.Errorf("manifest %s is missing required fields", path)
	}
	return manifest, nil
}
