# Contributing to Trainer

Thanks for helping. This is a small Go project and the loop is short.

## What you need

- Go (the version in [`go.mod`](go.mod)).
- `golangci-lint` for the lint step.
- `npx` (Node) if you want to exercise the add/delete/update commands, which
  shell out to `npx skills`. Browsing and the tests do not need it.

## Build and run

```sh
make build      # go build ./...
make run        # go run ./cmd/trainer
```

## Before you open a pull request

Run the full check. It must pass:

```sh
make verify     # gofmt check, go vet, go test, golangci-lint
```

## How the code is built

Trainer is built test-first in vertical slices. Each change cuts end to end,
from disk through logic to the screen, and is proven by a test that drives a
package's public entry point against a real temporary directory, not by a test
of an internal struct. Write the failing test first, then the smallest code
that passes it. The implementation plan and its slice history live in
[`docs/plans`](docs/plans); the project's vocabulary is in
[`CONTEXT.md`](CONTEXT.md). Reading both before a change keeps names and
behavior consistent.

## Pull requests

- One focused change per pull request.
- Keep the test that proves your change; a change to behavior needs a test that
  fails without it.
- Match the surrounding style. `make verify` covers formatting and linting.
