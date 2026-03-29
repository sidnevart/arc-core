package provider

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveBinary(binary string) (string, []string, error) {
	name := strings.TrimSpace(binary)
	if name == "" {
		return "", nil, fmt.Errorf("binary name is required")
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil, nil
	}

	notes := []string{"binary not found in PATH"}
	for _, dir := range binarySearchDirs() {
		candidate := filepath.Join(dir, name)
		notes = append(notes, "checked "+candidate)
		if executableFileExists(candidate) {
			return candidate, []string{"resolved outside PATH via known binary directories", "using " + candidate}, nil
		}
	}
	return "", notes, fmt.Errorf("%s not found in PATH or known binary locations", name)
}

func binarySearchDirs() []string {
	seen := map[string]bool{}
	out := []string{}
	appendDir := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, value)
	}

	for _, dir := range filepath.SplitList(os.Getenv("ARC_PROVIDER_BIN_DIRS")) {
		appendDir(dir)
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		appendDir(dir)
	}
	if runtime.GOOS == "darwin" {
		appendDir("/opt/homebrew/bin")
		appendDir("/usr/local/bin")
		appendDir("/opt/local/bin")
	}
	return out
}

func executableFileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func providerCommandEnv(resolvedBinary string, extraEnv ...string) []string {
	base := make([]string, 0, len(os.Environ())+len(extraEnv)+1)
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "PATH=") {
			continue
		}
		base = append(base, item)
	}
	pathValue := mergedProviderPath(resolvedBinary)
	if pathValue != "" {
		base = append(base, "PATH="+pathValue)
	}
	base = append(base, extraEnv...)
	return dedupeEnv(base)
}

func mergedProviderPath(resolvedBinary string) string {
	seen := map[string]bool{}
	out := []string{}
	appendDir := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, value)
	}

	appendDir(filepath.Dir(strings.TrimSpace(resolvedBinary)))
	for _, dir := range binarySearchDirs() {
		appendDir(dir)
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		appendDir(dir)
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func dedupeEnv(items []string) []string {
	lastByKey := map[string]string{}
	order := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		key := item
		if idx := strings.IndexByte(item, '='); idx >= 0 {
			key = item[:idx]
		}
		if _, seen := lastByKey[key]; !seen {
			order = append(order, key)
		}
		lastByKey[key] = item
	}
	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, lastByKey[key])
	}
	return out
}
