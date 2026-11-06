package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const usage = `treehash prints a deterministic SHA-256 for a directory tree.

Usage:
  treehash [flags] [path]

Flags:
  -exclude string  comma-separated names to skip (default ".git,node_modules,dist")
  -json            print a JSON report

Examples:
  treehash
  treehash ./src
  treehash -exclude ".git,node_modules,dist,coverage" -json .

Exit codes:
  0  the tree was hashed
  2  usage or I/O error
`

type report struct {
	Root  string `json:"root"`
	Files int    `json:"files"`
	Hash  string `json:"hash"`
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("treehash", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprint(stderr, usage)
	}

	exclude := flags.String("exclude", ".git,node_modules,dist", "comma-separated names to skip")
	asJSON := flags.Bool("json", false, "print a JSON report")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}

		return 2
	}

	if flags.NArg() > 1 {
		fmt.Fprintln(stderr, "treehash: at most one path is allowed")
		return 2
	}

	root := "."
	if flags.NArg() == 1 {
		root = flags.Arg(0)
	}

	result, err := hashTree(root, parseExclude(*exclude))
	if err != nil {
		fmt.Fprintf(stderr, "treehash: %v\n", err)
		return 2
	}

	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(stderr, "treehash: %v\n", err)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "%s  %s  (%d files)\n", result.Hash, result.Root, result.Files)
	}

	return 0
}

func parseExclude(value string) map[string]bool {
	excluded := make(map[string]bool)

	for _, name := range strings.Split(value, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			excluded[name] = true
		}
	}

	return excluded
}

func hashTree(root string, excluded map[string]bool) (report, error) {
	info, err := os.Stat(root)
	if err != nil {
		return report{}, fmt.Errorf("stat %s: %w", root, err)
	}
	if !info.IsDir() {
		return report{}, fmt.Errorf("%s is not a directory", root)
	}

	var files []string

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			if path != root && excluded[entry.Name()] {
				return filepath.SkipDir
			}

			return nil
		}

		if excluded[entry.Name()] || !entry.Type().IsRegular() {
			return nil
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		return report{}, fmt.Errorf("walk %s: %w", root, err)
	}

	sort.Strings(files)

	manifestLines := make([]string, 0, len(files))
	for _, path := range files {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return report{}, fmt.Errorf("relative %s: %w", path, err)
		}

		fileHash, err := hashFile(path)
		if err != nil {
			return report{}, err
		}

		manifestLines = append(manifestLines, fileHash+"  "+filepath.ToSlash(relative))
	}

	sort.Strings(manifestLines)

	manifest := sha256.New()
	for _, line := range manifestLines {
		if _, err := io.WriteString(manifest, line+"\n"); err != nil {
			return report{}, fmt.Errorf("hash manifest: %w", err)
		}
	}

	return report{
		Root:  root,
		Files: len(files),
		Hash:  hex.EncodeToString(manifest.Sum(nil)),
	}, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}

	hasher := sha256.New()
	_, copyErr := io.Copy(hasher, file)
	closeErr := file.Close()

	if copyErr != nil {
		return "", fmt.Errorf("read %s: %w", path, copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close %s: %w", path, closeErr)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
