set shell := ["bash", "-cu"]

binary := "podji"

default:
  just --list

build:
  @version=$(git describe --tags --always --dirty 2>/dev/null || echo dev); \
  commit=$(git rev-parse --short HEAD 2>/dev/null || echo none); \
  date=$(date -u +%Y-%m-%dT%H:%M:%SZ); \
  go build \
    -ldflags "-X github.com/dloss/podji/internal/buildinfo.Version=$version -X github.com/dloss/podji/internal/buildinfo.Commit=$commit -X github.com/dloss/podji/internal/buildinfo.Date=$date" \
    -o {{binary}} ./cmd/podji

run: build
  ./{{binary}}

run-dev:
  go run ./cmd/podji

test:
  go test ./...

test-race:
  @for p in $(go list ./internal/...); do \
    go test -race "$p"; \
  done

fmt:
  gofmt -w .

fmt-check:
  @unformatted=$(gofmt -l .); \
  if [ -n "$unformatted" ]; then \
    echo "The following files are not gofmt-formatted:"; \
    echo "$unformatted"; \
    exit 1; \
  fi

vet:
  go vet ./...

vuln:
  @if ! command -v govulncheck >/dev/null 2>&1; then \
    echo "Installing govulncheck..."; \
    go install golang.org/x/vuln/cmd/govulncheck@latest; \
  fi
  @PATH="$(go env GOPATH)/bin:$$PATH" govulncheck ./...

check: fmt-check vet test

ci: check test-race vuln

clean:
  rm -f {{binary}}

ui-start:
  dev/ui.sh start

ui-cap:
  dev/ui.sh cap

ui-quit:
  dev/ui.sh quit
