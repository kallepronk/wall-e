# AGENTS.md — Walle Codebase Guide

WALL-E is a Go CLI tool that removes comments from codebases. It uses
`go-tree-sitter` to parse source files across 27+ languages and targets
git-tracked changes (diffs, commit ranges, or specific files).

---

## Project Structure

```
walle/
├── main.go                  # Entrypoint — calls cmd.Execute()
├── go.mod / go.sum          # Module name: "walle", Go 1.25+
├── Makefile                 # Currently empty
├── tests/
│   └── testdata/            # Fixture files for future integration tests (one per language)
└── internal/
    ├── cmd/                 # Cobra CLI commands (root.go, scan.go, fix.go)
    ├── comment/             # Core logic: model.go, scanner.go, remover.go
    ├── discovery/           # Git-based file discovery (scanner, diff, history, utils)
    ├── languages/           # Language registry: extension → tree-sitter grammar
    └── pipeline/            # Concurrent scan orchestration
```

---

## Build, Lint, and Test Commands

```bash
# Build the binary
go build -o walle .

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests in a specific package
go test ./internal/comment/...

# Run a single named test
go test ./internal/comment/ -run TestFunctionName

# Run with race detector
go test -race ./...

# Vet (static analysis)
go vet ./...

# Format all source files
gofmt -w .

# Format + fix imports
goimports -w .
```

> Note: The `Makefile` is currently empty. There are no test files (`*_test.go`)
> yet — `tests/testdata/` holds fixture source files intended for future tests.

---

## Code Style

Follow standard Go conventions. No linter config files exist; `gofmt` and
`go vet` are the baseline. The `.idea/go.imports.xml` explicitly excludes two
deprecated packages — **do not use them**:

- `github.com/pkg/errors` — use stdlib `errors` / `fmt.Errorf` with `%w`
- `golang.org/x/net/context` — use stdlib `context`

### Formatting

- Tabs for indentation (enforced by `gofmt`).
- Opening braces on the same line as the statement.
- One blank line between top-level declarations.
- No trailing whitespace; no semicolons.

---

## Imports

Use **two-group import style**: stdlib first, then external packages, separated
by a blank line. Internal module paths (`walle/internal/...`) are grouped with
stdlib, not with external packages.

```go
import (
    "context"
    "fmt"
    "path/filepath"
    "strings"
    "walle/internal/discovery"
    "walle/internal/languages"

    sitter "github.com/smacker/go-tree-sitter"
)
```

- Alias imports only when necessary to avoid a name collision or for brevity
  (e.g., `sitter` for `go-tree-sitter`).
- No dot imports (`. "pkg"`).
- No blank imports (`_ "pkg"`) unless required for side-effects (e.g., driver
  registration).

---

## Naming Conventions

### Files

Lowercase, snake_case for multi-word names: `git_diff.go`, `git_history.go`,
`specific_file.go`. One-concept files use the concept name directly:
`scanner.go`, `remover.go`, `pipeline.go`. Each package has a `model.go` for
its data types.

### Packages

Short, lowercase, single words matching the directory name: `cmd`, `comment`,
`discovery`, `languages`, `pipeline`.

### Types and Structs

PascalCase, noun-first:

```go
Comment, File, LineRange, ScanOptions, ScanType, FileStatus
GitScanner, TreeSitterScanner, LanguageConfig
```

Use `Options` within a package context (callers use the full `pipeline.Options`).

### Interfaces

PascalCase, ending in `-er` per Go convention: `Scanner`, `Collect`.

### Constants and Enum-Like Values

PascalCase with a type prefix:

```go
type ScanType int
const (
    ScanWhole ScanType = iota
    ScanDiff
)

type FileStatus string
const (
    StatusModified   FileStatus = "modified"
    StatusAdded      FileStatus = "added"
    StatusUntracked  FileStatus = "untracked"
    StatusDeleted    FileStatus = "deleted"
)
```

### Functions and Methods

- Exported: PascalCase — `Execute`, `GetScanner`, `RemoveComments`,
  `ScanPipeline`, `FromGitDiff`, `ValidateCommitOrder`.
- Unexported: camelCase — `runScan`, `runFix`, `findAllFiles`,
  `isSupportedFile`, `getRepoRoot`, `computeLCS`.

### Variables

camelCase. Cobra flag variables at package level follow the pattern
`<command><FlagName>`: `scanAll`, `scanPath`, `fixBaseCommit`,
`fixIgnoreGitIgnore`. Loop and temporary variables use short names:
`i`, `j`, `r`, `gi`, `wg`, `mu`.

### CLI Commands

Lowercase verb nouns matching the Cobra `Use` field: `scan`, `fix`.

---

## Error Handling

Use stdlib only — no `github.com/pkg/errors`.

**Wrap errors with context using `fmt.Errorf` + `%w`** (primary pattern):

```go
return nil, fmt.Errorf("failed to get worktree: %w", err)
return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
```

**Sentinel errors via `errors.New`** for non-wrappable situations:

```go
return nil, errors.New("no git repository found (are you in a git repo?)")
```

**CLI command error output** — print to stdout with `fmt.Printf`, return early
(do not call `os.Exit` inside subcommand `Run` functions):

```go
if err != nil {
    fmt.Printf("Error scanning: %v\n", err)
    return
}
```

**Silent/graceful degradation** for non-critical operations (e.g., loading
`.gitignore`):

```go
gi, err := gitignore.CompileIgnoreFile(gitignorePath)
if err != nil {
    return nil  // caller handles nil check
}
```

**Guard clauses / early returns** for empty inputs:

```go
if len(comments) == 0 {
    return nil
}
```

Error message format: `"failed to <verb> <noun>: %w"` — lowercase, no trailing
period, wrapped with `%w` for unwrapping.

---

## Concurrency

`ScanPipeline` (`internal/pipeline/pipeline.go`) uses a goroutine-per-file
fan-out pattern with `sync.WaitGroup` + `sync.Mutex` to protect shared state.
Follow this pattern when adding parallelism — avoid channels for simple
fan-out/collect unless the pipeline naturally benefits from streaming.

---

## Key Design Patterns

- **`model.go` per package** — keep data types separate from logic files.
- **`From*` factory functions** for `Collect` function factories
  (`FromGitDiff`, `FromGitHistory`, `FromSpecificFile`).
- **`Scanner` interface** used in both `comment` and `discovery` packages to
  allow swapping implementations.
- **`internal/` for all non-entrypoint code** — nothing outside `main.go`
  lives at the module root.

---

## Notes for Agents

- `internal/cmd/scan.go` currently has a syntax error (incomplete assignment on
  an in-progress refactor). Fix this before running `go build`.
- `ignore.go` at the repo root is gitignored and used as a scratch file — do
  not modify or commit it.
- The compiled `walle` binary is committed to the repo root — rebuild after
  making changes with `go build -o walle .`.
- No `.cursorrules`, `.github/copilot-instructions.md`, or CI workflow files
  exist in this repository.
