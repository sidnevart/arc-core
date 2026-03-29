package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-os/internal/project"
)

type UploadedProjectFile struct {
	Name          string `json:"name"`
	ContentBase64 string `json:"content_base64"`
	MimeType      string `json:"mime_type,omitempty"`
	Size          int64  `json:"size,omitempty"`
}

func (s Service) ProjectMaterials(root string) ([]ProjectMaterialSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return nil, err
	}
	items, err := loadProjectMaterials(resolved)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UploadedAt > items[j].UploadedAt
	})
	return items, nil
}

func (s Service) UploadProjectMaterials(root string, sessionID string, files []UploadedProjectFile) ([]ProjectMaterialSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	destDir := project.ProjectFile(resolved, "materials", "uploads", stamp)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}

	items, err := loadProjectMaterials(resolved)
	if err != nil {
		return nil, err
	}
	created := make([]ProjectMaterialSummary, 0, len(files))
	for index, file := range files {
		name := sanitizeUploadName(file.Name)
		if name == "" {
			name = fmt.Sprintf("upload-%d.txt", index+1)
		}
		content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(file.ContentBase64))
		if err != nil {
			return nil, fmt.Errorf("decode upload %q: %w", file.Name, err)
		}
		path := uniqueUploadPath(destDir, name)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return nil, err
		}
		rel, _ := filepath.Rel(resolved, path)
		item := ProjectMaterialSummary{
			ID:               fmt.Sprintf("%s-%d", stamp, index+1),
			Name:             filepath.Base(path),
			Path:             filepath.ToSlash(rel),
			Size:             int64(len(content)),
			UploadedAt:       time.Now().UTC().Format(time.RFC3339),
			Source:           "ui_upload",
			RelatedSessionID: strings.TrimSpace(sessionID),
			MimeType:         strings.TrimSpace(file.MimeType),
		}
		items = append(items, item)
		created = append(created, item)
	}
	if err := saveProjectMaterials(resolved, items); err != nil {
		return nil, err
	}
	return created, nil
}

func (s Service) DeleteProjectMaterial(root string, materialID string) ([]ProjectMaterialSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return nil, err
	}
	items, err := loadProjectMaterials(resolved)
	if err != nil {
		return nil, err
	}
	materialID = strings.TrimSpace(materialID)
	if materialID == "" {
		return nil, fmt.Errorf("material id is required")
	}
	filtered := make([]ProjectMaterialSummary, 0, len(items))
	removed := false
	for _, item := range items {
		if item.ID != materialID {
			filtered = append(filtered, item)
			continue
		}
		removed = true
		if err := removeManagedMaterialFile(resolved, item.Path); err != nil {
			return nil, err
		}
	}
	if !removed {
		return nil, fmt.Errorf("project material %q not found", materialID)
	}
	if err := saveProjectMaterials(resolved, filtered); err != nil {
		return nil, err
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UploadedAt > filtered[j].UploadedAt
	})
	return filtered, nil
}

func loadProjectMaterials(root string) ([]ProjectMaterialSummary, error) {
	path := project.ProjectFile(root, "materials", "uploads.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return []ProjectMaterialSummary{}, nil
		}
		return nil, err
	}
	var items []ProjectMaterialSummary
	if err := project.ReadJSON(path, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func saveProjectMaterials(root string, items []ProjectMaterialSummary) error {
	return project.WriteJSON(project.ProjectFile(root, "materials", "uploads.json"), items)
}

func sanitizeUploadName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "..", "")
	return name
}

func uniqueUploadPath(dir string, name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	candidate := filepath.Join(dir, name)
	index := 1
	for {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, index, ext))
		index++
	}
}

func removeManagedMaterialFile(root string, relPath string) error {
	cleanRel := filepath.Clean(strings.TrimSpace(relPath))
	if cleanRel == "." || cleanRel == "" {
		return fmt.Errorf("material path is required")
	}
	arcRoot := project.ProjectFile(root)
	fullPath := filepath.Join(root, cleanRel)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return err
	}
	absArcRoot, err := filepath.Abs(arcRoot)
	if err != nil {
		return err
	}
	if absPath != absArcRoot && !strings.HasPrefix(absPath, absArcRoot+string(os.PathSeparator)) {
		return fmt.Errorf("project material %q is not ARC-managed", relPath)
	}
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	for dir := filepath.Dir(absPath); dir != absArcRoot && dir != "." && dir != string(os.PathSeparator); dir = filepath.Dir(dir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			break
		}
		if err := os.Remove(dir); err != nil {
			break
		}
	}
	return nil
}
