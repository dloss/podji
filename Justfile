set shell := ["bash", "-cu"]

binary := "podji"

default:
  just --list

build:
  go build -o {{binary}} ./cmd/podji

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
