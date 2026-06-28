# AGENTS.md

Guidance for AI coding agents working in the `hd-idle` repository.

## Project Overview

`hd-idle` is a Go utility that spins down idle hard disks after a period of
inactivity. It monitors `/proc/diskstats`, tracks read/write activity per
device, and issues SCSI or ATA spindown commands via the SG_IO ioctl interface.
Linux only.

## Repository Layout

- `main.go` — CLI argument parsing, entry point, run loop.
- `hdidle.go` — Core monitoring logic, config types, spindown orchestration.
- `diskstats/` — Parses `/proc/diskstats`; classifies devices as
  Disk/Partition/DeviceMapper and resolves holders (for LUKS support).
- `sgio/` — SG_IO ioctl wrappers for SCSI START STOP UNIT and ATA PASS-THROUGH
  spindown; detects JMicron USB-SATA bridge controllers.
- `io/` — Symlink resolution helpers (`RealPath`).
- `vendor/` — Vendored dependencies (do not edit; `go mod vendor` managed).
- `debian/` — Debian packaging files (rules, service, man page).

## Build / Test / Lint Commands

### Build / install

```sh
make                 # builds ./hd-idle for current arch (sets GOOS=linux)
go build             # direct build (no arch pinning)
make install         # installs to /usr/local (or $DESTDIR/usr for Debian)
```

### Run all tests (matches Makefile `test` target)

```sh
make test
# equivalent:
GO111MODULE=on go test ./... -race -cover
```

### Run a single test

```sh
go test -run TestIntervalWith300SecondsIdle .          # root package
go test -run TestStatsForDisk/disk_type ./diskstats/   # single subtest
go test -run TestRealPath ./io/                         # subpackage
go test -run TestAtaDevice_deviceType ./sgio/
```

### Lint / format

```sh
go vet ./...     # passes clean — keep it that way
gofmt -l .       # list files needing formatting (should be empty)
gofmt -w .       # fix formatting
```

`gofmt` is the authority. There is no `gofmt`/`golangci-lint` config file in
the repo; match what `gofmt` produces. Do not introduce a linter config without
asking.

## Code Style

### Formatting

- Tabs for indentation (gofmt default). No trailing whitespace.
- Run `gofmt -w` before committing; CI/Makefile does not enforce it but the
  codebase is gofmt-clean (treat `gofmt -l .` output as a build failure).

### Imports

- Single `import (...)` block per file, **alphabetically sorted across stdlib
  and third-party together** (this is the existing convention — see `main.go`,
  `hdidle.go`). Do not split into separate stdlib/third-party groups; that would
  diverge from the rest of the codebase.
- Internal packages are referenced by their full path
  (`github.com/adelolmo/hd-idle/diskstats`, etc.).

### File header

Every non-vendored `.go` file begins with the GPL v3 license header:

```go
// hd-idle - spin down idle hard disks
// Copyright (C) 2018  Andoni del Olmo
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
```

Add this header to any new `.go` file in the main module (not `vendor/`).

### Naming

- Exported identifiers: `PascalCase` (e.g. `ObserveDiskActivity`, `DiskStats`).
- Unexported: `camelCase` (e.g. `poolInterval`, `previousSnapshots`).
- Receiver names are short, often 1-2 letters: `c` for `Config`, `dc` for
  `DeviceConf`, `ad` for `AtaDevice`, `a` for `apt`.
- Constants: grouped in `const (...)` blocks; exported string constants are
  uppercase (`SCSI`, `ATA`), unexported are `camelCase` (`dateFormat`,
  `startStopUnit`).

### Types

- Plain structs; no interfaces are defined (functions accept concrete types or
  function types like `diskHolderGetterFunc`).
- `iota` used for enum-like types (`DeviceType`, JMicron/Unknown).
- No generics in use (module declares `go 1.16`).

### Error handling

- Functions that can fail return `(value, error)`.
- Wrap errors with context using `fmt.Errorf("cannot spindown ... %s", err)`
  (note: this codebase predates `%w` — follow the existing `err.Error()` or
  `%s` style rather than introducing `%w` unless rewriting a whole error path).
- User-facing CLI errors: `fmt.Println(...)` then `os.Exit(1)` (see `main.go`).
- Fatal runtime errors: `log.Fatal` / `log.Fatalf` (see `logToFile`,
  `diskstats.Snapshot`).
- Non-fatal spindown errors are printed and the loop continues
  (`hdidle.go:156`); do not exit the run loop on a single device failure.

### Logging / output

- `fmt.Print*` for stdout (spin up/down events, debug, usage); `logToFile`
  appends to the optional `-l` logfile only when a path is set. Debug output is
  gated behind `config.Defaults.Debug` / the `debug` bool arg.

### Globals

- `previousSnapshots`, `now`, `lastNow` are package-level globals in `hdidle.go`
  (package `main`). The monitoring loop is single-goroutine; tests rely on this.
  Do not introduce concurrent writers without refactoring these into a struct.

## Testing Conventions

- Tests live alongside source as `*_test.go` in the same package (white-box).
- Table-driven tests are the norm: define a slice of anonymous structs with
  `name` + inputs + `want`, iterate with `t.Run(test.name, func(t *testing.T){...})`.
- Assertions are manual using `t.Fatalf` (stop) / `t.Errorf` (continue) with
  `Expected %v but found %v` style messages.
- Filesystem-touching tests use `/tmp` paths and `os.RemoveAll` +
  `os.MkdirAll`/`os.WriteFile`/`os.Symlink` to set up fixtures; clean up at the
  start of each case. There is no `t.TempDir()` usage — follow the existing
  `/tmp/...` fixture pattern for consistency.
- The Makefile runs tests with `-race -cover`; keep new code race-clean.

## Vendor / Dependencies

- Single external dependency: `github.com/benmcclelland/sgio` (vendored).
- Do not add new dependencies without strong justification; this is a
  low-dependency systems utility. Run `go mod tidy && go mod vendor` if modules
  change.

## Things to Avoid

- Do not split import groups (see Imports above).
- Do not add comments restating what code does; comments exist only for
  non-obvious logic (e.g. kernel diskstats field layout, ATA opcodes, LUKS
  holder behavior).
- Do not change `go.mod`'s `go 1.16` directive without reason — bumping it
  affects minimum supported toolchains.
- Do not edit files under `vendor/` directly.
- Do not introduce `context.Context` plumbing unless a specific need arises;
  the daemon is a simple blocking loop and current APIs do not take contexts.
