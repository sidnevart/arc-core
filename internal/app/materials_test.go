package app

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"agent-os/internal/project"
)

func TestUploadProjectMaterialsStoresFilesAndIndex(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	svc := Service{}
	payload := base64.StdEncoding.EncodeToString([]byte("hello from upload"))
	created, err := svc.UploadProjectMaterials(root, "session-123", []UploadedProjectFile{
		{
			Name:          "notes.md",
			ContentBase64: payload,
			MimeType:      "text/markdown",
			Size:          int64(len("hello from upload")),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 created material, got %d", len(created))
	}
	if created[0].Path == "" {
		t.Fatal("expected created material path")
	}
	if got, want := filepath.ToSlash(created[0].Path), ".arc/materials/uploads/"; len(got) <= len(want) || got[:len(want)] != want {
		t.Fatalf("expected ARC-managed upload path, got %q", created[0].Path)
	}
	if created[0].RelatedSessionID != "session-123" {
		t.Fatalf("expected session link, got %q", created[0].RelatedSessionID)
	}

	content, err := os.ReadFile(filepath.Join(root, created[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello from upload" {
		t.Fatalf("unexpected file content: %q", string(content))
	}

	items, err := svc.ProjectMaterials(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 indexed material, got %d", len(items))
	}
	if items[0].Name != "notes.md" {
		t.Fatalf("expected material name notes.md, got %q", items[0].Name)
	}
}

func TestDeleteProjectMaterialRemovesManagedFileAndIndex(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	svc := Service{}
	payload := base64.StdEncoding.EncodeToString([]byte("delete me"))
	created, err := svc.UploadProjectMaterials(root, "session-123", []UploadedProjectFile{{
		Name:          "delete-me.md",
		ContentBase64: payload,
		MimeType:      "text/markdown",
		Size:          int64(len("delete me")),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 created material, got %d", len(created))
	}

	if _, err := os.Stat(filepath.Join(root, created[0].Path)); err != nil {
		t.Fatalf("expected upload to exist before deletion: %v", err)
	}

	items, err := svc.DeleteProjectMaterial(root, created[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty registry after deletion, got %d item(s)", len(items))
	}
	if _, err := os.Stat(filepath.Join(root, created[0].Path)); !os.IsNotExist(err) {
		t.Fatalf("expected upload file to be removed, got %v", err)
	}
}
