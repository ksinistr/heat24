# CRUSH.md for heat24

## Commands

### Build & Run
- Build: `go build -o heat24 ./cmd`
- Run: `go run ./cmd`
- Clean: `go clean`

### Test
- All: `go test -v ./...`
- Single: `go test -v -run ^TestName$ ./internal/...` (no tests currently)

### Lint & Check
- Format: `go fmt ./...`
- Vet: `go vet ./...`
- Type check: `go vet ./...` (uses Go's built-in)

## Code Style Guidelines

### Imports
- Group: Standard library (e.g., fmt, time), then third-party (e.g., gonum), then local (internal).
- Single import per line: `import "fmt"`.
- No aliases unless necessary.

### Formatting
- Use `go fmt` for standard indentation (tabs), 8-space shift width.
- Line length: Flexible, but prefer <100 chars.
- No trailing whitespace.

### Types & Naming
- Exported: Uppercase (e.g., `type Averages struct`).
- Internal: Lowercase (e.g., `func heatIndexC`).
- Structs: Simple fields (e.g., `TempSum float64`).
- Vars/Funcs: Descriptive, camelCase.

### Error Handling
- Standard: `if err != nil { return ..., err }`.
- Main: `log.Fatalf("msg: %v", err)`.
- No panics; propagate errors.

### General
- No comments in code (self-documenting).
- Use math.NaN for missing values.
- Dependencies: Stick to go.mod (gonum/plot, httpcache, etc.).
- README outdated (Python refs); update for Go.

## Notes
- Project fetches Open-Meteo weather, computes heat index, plots PDFs.
- No Cursor/Copilot rules found.