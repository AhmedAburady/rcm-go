binary := "rcm"
module := "github.com/AhmedAburady/rcm-go"

version    := `git describe --tags --always --dirty 2>/dev/null || echo dev`
commit     := `git rev-parse --short HEAD 2>/dev/null || echo none`
build_date := `date -u +"%Y-%m-%dT%H:%M:%SZ"`

ldflags := "-X " + module + "/internal/cmd.Version=" + version + \
           " -X " + module + "/internal/cmd.Commit=" + commit + \
           " -X " + module + "/internal/cmd.BuildDate=" + build_date

# List available recipes
default:
    @just --list

# Build the binary with version info stamped in
build:
    go build -ldflags "{{ldflags}}" -o {{binary}} ./cmd/rcm

# Quick development build (no ldflags, faster)
dev:
    go build -o {{binary}} ./cmd/rcm

# Install to GOPATH/bin with version info
install:
    go install -ldflags "{{ldflags}}" ./cmd/rcm

# Run tests
test:
    go test ./...

# Run tests with an HTML coverage report
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Vet, format-check, and test
check: fmt-check
    go vet ./...
    go test ./...

# Format code
fmt:
    go fmt ./...

# Fail if any file is not gofmt-clean
fmt-check:
    @test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)

# Tidy go modules
tidy:
    go mod tidy

# Apply safe Go modernizations
fix:
    go fix ./...

# Remove build artifacts
clean:
    rm -f {{binary}} {{binary}}-*
    rm -f coverage.out coverage.html

# Cross-compile for all release platforms
build-all: (cross "linux" "amd64") (cross "linux" "arm64") (cross "darwin" "amd64") (cross "darwin" "arm64")

# Cross-compile for one GOOS/GOARCH, e.g. `just cross linux arm64`
cross goos goarch:
    GOOS={{goos}} GOARCH={{goarch}} go build -ldflags "{{ldflags}}" -o {{binary}}-{{goos}}-{{goarch}} ./cmd/rcm
