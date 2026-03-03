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
  go test -race ./...

clean:
  rm -f {{binary}}

ui-start:
  dev/ui.sh start

ui-cap:
  dev/ui.sh cap

ui-quit:
  dev/ui.sh quit
