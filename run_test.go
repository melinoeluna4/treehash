package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHashTreeDeterministic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "a.txt", "alpha")
	writeTestFile(t, root, "nested/b.txt", "beta")

	first, err := hashTree(root, map[string]bool{})
	if err != nil {
		t.Fatalf("first hashTree: %v", err)
	}

	second, err := hashTree(root, map[string]bool{})
	if err != nil {
		t.Fatalf("second hashTree: %v", err)
	}

	if first.Hash != second.Hash {
		t.Fatalf("hashes differ: %s vs %s", first.Hash, second.Hash)
	}
	if first.Files != 2 {
		t.Fatalf("files = %d, want 2", first.Files)
	}
}

func TestHashTreeChangesWithContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "a.txt", "alpha")

	before, err := hashTree(root, map[string]bool{})
	if err != nil {
		t.Fatalf("before hashTree: %v", err)
	}

	writeTestFile(t, root, "a.txt", "changed")

	after, err := hashTree(root, map[string]bool{})
	if err != nil {
		t.Fatalf("after hashTree: %v", err)
	}

	if before.Hash == after.Hash {
		t.Fatal("hash did not change after modifying a file")
	}
}

func TestHashTreeExcludesNames(t *testing.T) {
	t.Parallel()

	withIgnored := t.TempDir()
	writeTestFile(t, withIgnored, "a.txt", "alpha")
	writeTestFile(t, withIgnored, "node_modules/pkg/index.js", "ignored")

	withoutIgnored := t.TempDir()
	writeTestFile(t, withoutIgnored, "a.txt", "alpha")

	first, err := hashTree(withIgnored, parseExclude("node_modules"))
	if err != nil {
		t.Fatalf("withIgnored hashTree: %v", err)
	}

	second, err := hashTree(withoutIgnored, parseExclude("node_modules"))
	if err != nil {
		t.Fatalf("withoutIgnored hashTree: %v", err)
	}

	if first.Hash != second.Hash {
		t.Fatalf("hashes differ after exclusion: %s vs %s", first.Hash, second.Hash)
	}
	if first.Files != 1 {
		t.Fatalf("files = %d, want 1", first.Files)
	}
}

func TestHashTreeRejectsFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := writeTestFile(t, root, "a.txt", "alpha")

	if _, err := hashTree(path, map[string]bool{}); err == nil {
		t.Fatal("hashTree accepted a file as root")
	}
}

func TestRunJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "a.txt", "alpha")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-json", root}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0; stderr = %s", code, stderr.String())
	}

	var result report
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v; output = %s", err, stdout.String())
	}
	if result.Hash == "" || result.Files != 1 {
		t.Fatalf("result = %#v, want one hashed file", result)
	}
}

func TestParseExclude(t *testing.T) {
	t.Parallel()

	excluded := parseExclude(" .git, node_modules, , dist ")
	for _, name := range []string{".git", "node_modules", "dist"} {
		if !excluded[name] {
			t.Fatalf("excluded[%q] = false, want true", name)
		}
	}
}

func writeTestFile(t *testing.T, root, name, content string) string {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	return path
}
