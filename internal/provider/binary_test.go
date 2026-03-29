package provider

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveBinaryFallsBackToConfiguredDirs(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "codex")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARC_PROVIDER_BIN_DIRS", tmp)
	t.Setenv("PATH", "")

	resolved, notes, err := resolveBinary("codex")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != path {
		t.Fatalf("expected %q, got %q", path, resolved)
	}
	if len(notes) == 0 {
		t.Fatal("expected fallback notes")
	}
}

func TestRunProviderCommandExtendsPathForShebangDependencies(t *testing.T) {
	tmp := t.TempDir()
	nodePath := filepath.Join(tmp, "node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\nprintf 'ok-from-node\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	codexPath := filepath.Join(tmp, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/usr/bin/env node\nconsole.log('ignored')\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ARC_PROVIDER_BIN_DIRS", tmp)
	t.Setenv("PATH", "")

	stdoutPath := filepath.Join(tmp, "provider.stdout.log")
	stderrPath := filepath.Join(tmp, "provider.stderr.log")
	stdout, stderr, err, timedOut := runProviderCommand("codex", nil, tmp, time.Second, stdoutPath, stderrPath)
	if timedOut {
		t.Fatal("expected provider command to finish without timeout")
	}
	if err != nil {
		t.Fatalf("expected provider command to succeed, got %v (stderr=%q)", err, string(stderr))
	}
	if !bytes.Contains(stdout, []byte("ok-from-node")) {
		t.Fatalf("expected shebang dependency to resolve via PATH, got stdout=%q stderr=%q", string(stdout), string(stderr))
	}
}

func TestProviderCommandEnvLetsLaterOverridesWin(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://example.com")
	env := providerCommandEnv("/tmp/codex", "OTEL_EXPORTER_OTLP_ENDPOINT=", "OTEL_TRACES_EXPORTER=none")
	values := map[string]string{}
	for _, item := range env {
		parts := strings.SplitN(item, "=", 2)
		key := parts[0]
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		values[key] = value
	}
	if got := values["OTEL_EXPORTER_OTLP_ENDPOINT"]; got != "" {
		t.Fatalf("expected OTEL_EXPORTER_OTLP_ENDPOINT to be cleared, got %q", got)
	}
	if got := values["OTEL_TRACES_EXPORTER"]; got != "none" {
		t.Fatalf("expected OTEL_TRACES_EXPORTER=none, got %q", got)
	}
}
