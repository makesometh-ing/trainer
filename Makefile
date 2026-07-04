.PHONY: build test vet lint fmt fmt-check run verify tidy release-check release-snapshot

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -w cmd internal

fmt-check:
	@test -z "$$(gofmt -l cmd internal)" || (echo "gofmt needed:"; gofmt -l cmd internal; exit 1)

run:
	go run ./cmd/trainer

tidy:
	go mod tidy

verify: fmt-check vet test lint

# Requires goreleaser (https://goreleaser.com). Not run by `verify`.
release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean
