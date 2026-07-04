package fswalk

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		"src",
		"src/main",
		"vendor",
		"vendor/lib",
		"node_modules",
		"node_modules/pkg",
	}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}

	files := []string{
		"README.md",
		"src/main/app.go",
		"src/main/app_test.go",
		"src/main/data.json",
		"vendor/lib/dep.go",
		"node_modules/pkg/index.js",
	}
	for _, f := range files {
		os.WriteFile(filepath.Join(root, f), []byte("test"), 0o644)
	}
	return root
}

func collectPaths(root string, opts *Options) ([]string, error) {
	var mu sync.Mutex
	var paths []string
	err := Walk(root, opts, func(path string, d fs.DirEntry) error {
		rel, _ := filepath.Rel(root, path)
		mu.Lock()
		paths = append(paths, filepath.ToSlash(rel))
		mu.Unlock()
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func TestWalkNoFilter(t *testing.T) {
	root := setupTestDir(t)
	paths, err := collectPaths(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 6 {
		t.Fatalf("expected 6 files, got %d: %v", len(paths), paths)
	}
}

func TestWalkSkipGlob(t *testing.T) {
	root := setupTestDir(t)
	paths, err := collectPaths(root, &Options{
		Skip: []string{"**/vendor/**", "**/node_modules/**"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if strings.Contains(p, "vendor") || strings.Contains(p, "node_modules") {
			t.Fatalf("should have skipped %s", p)
		}
	}
	if len(paths) != 4 {
		t.Fatalf("expected 4 files, got %d: %v", len(paths), paths)
	}
}

func TestWalkDenyExts(t *testing.T) {
	root := setupTestDir(t)
	paths, err := collectPaths(root, &Options{
		DenyExts: []string{".md", ".json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		ext := filepath.Ext(p)
		if ext == ".md" || ext == ".json" {
			t.Fatalf("should have denied %s", p)
		}
	}
}

func TestWalkAllowExts(t *testing.T) {
	root := setupTestDir(t)
	paths, err := collectPaths(root, &Options{
		AllowExts: []string{".go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if filepath.Ext(p) != ".go" {
			t.Fatalf("should only allow .go, got %s", p)
		}
	}
}

func TestWalkIncludeGlob(t *testing.T) {
	root := setupTestDir(t)
	paths, err := collectPaths(root, &Options{
		Include: []string{"**/*_test.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || !strings.HasSuffix(paths[0], "app_test.go") {
		t.Fatalf("expected only app_test.go, got %v", paths)
	}
}

func TestWalkCustomFilter(t *testing.T) {
	root := setupTestDir(t)
	paths, err := collectPaths(root, &Options{
		Filter: func(path string, d fs.DirEntry) bool {
			return !strings.HasSuffix(d.Name(), "_test.go")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			t.Fatalf("filter should have excluded %s", p)
		}
	}
}

func TestGlobFilterNil(t *testing.T) {
	var f *GlobFilter
	if f.Match("/some/path", false) {
		t.Fatal("nil filter should never match")
	}
}
