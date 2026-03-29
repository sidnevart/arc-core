package app

import (
	"testing"

	"agent-os/internal/project"
)

func TestDeveloperAccessDefaultsToUserWithoutConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	state, err := Service{}.DeveloperAccessState(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Role != "user" {
		t.Fatalf("expected user role, got %q", state.Role)
	}
	if state.CanUseTesting {
		t.Fatal("expected testing to be disabled by default")
	}
}

func TestSetDeveloperRolePersistsProjectConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	state, err := Service{}.SetDeveloperRole(root, "developer")
	if err != nil {
		t.Fatal(err)
	}
	if state.Role != "developer" || !state.CanUseTesting {
		t.Fatalf("unexpected developer access state: %+v", state)
	}

	state, err = Service{}.DeveloperAccessState(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Source != "project_config" {
		t.Fatalf("expected project_config source, got %q", state.Source)
	}
}
