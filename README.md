# treehash

> **A stable fingerprint for a directory tree.**

`treehash` is a small Go CLI that hashes every regular file in a directory, combines the relative paths and content hashes in sorted order, and prints one deterministic SHA-256. Use it for CI cache keys, deployment manifests or “did this directory change?” checks.

No dependencies outside the Go standard library.

## Why this exists

A timestamp is not a cache key. A Git commit is not enough when the build also depends on generated files. `treehash` gives the same hash for the same tree, regardless of walk order, line endings in file names or the machine that runs it.

## Install

```sh
go install github.com/melinoeluna4/treehash@latest
```

## Usage

```sh
treehash
```

```text
9c1d8f0e4b4f7c3b1f6a2d5e8c0b7a9d6e3f1c2b4a5d6e7f8a9b0c1d2e3f4a5b  .  (42 files)
```

Hash another directory, skip more names or emit JSON:

```sh
treehash ./src
treehash -exclude ".git,node_modules,dist,coverage" .
treehash -json .
```

### Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-exclude` | `.git,node_modules,dist` | comma-separated file or directory names to skip |
| `-json` | `false` | print a JSON report |

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | the tree was hashed |
| `2` | usage or I/O error |

## How it works

1. Walk the root with `filepath.WalkDir`.
2. Skip excluded file or directory names and non-regular files (symlinks, sockets, devices).
3. Hash each file with SHA-256.
4. Build a sorted manifest of `hash  relative/path` lines.
5. Hash the manifest with SHA-256 and print it.

The manifest includes the relative path, so renaming a file changes the tree hash even when its content does not.

## Development

```sh
go test ./...
go vet ./...
go build ./...
```

CI runs the same three commands on every push and pull request.

---

Part of the [Melinoe Luna portfolio](https://melinoeluna4.github.io/proyectos/treehash/) · Go 1.27 · standard library only.
