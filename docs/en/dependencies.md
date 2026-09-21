# Dependency Inventory

This document records every third-party dependency used by scli, per Rule 18.

---

## Direct dependencies

### `github.com/spf13/cobra`

| Field | Value |
|-------|-------|
| Version | v1.10.2 |
| License | Apache-2.0 |
| Purpose | CLI framework — command/subcommand dispatch, flag parsing, help generation |
| Why not in-house | Cobra is the de-facto standard for Go CLIs. Reimplementing subcommand routing, flag inheritance, help text generation, and shell completion would be substantial scope with no benefit. |

---

## Indirect dependencies

These are pulled in by direct dependencies. They are not imported directly by scli.

### `github.com/fatih/color`

| Field | Value |
|-------|-------|
| Version | v1.19.0 |
| License | MIT |
| Purpose | ANSI colour output; TTY auto-detection on all platforms |
| Why not in-house | Handles Windows ANSI emulation and isatty checks across platforms. Pulling this in saves ~200 lines of OS-specific code that is not core to scli. |

### `github.com/zalando/go-keyring`

| Field | Value |
|-------|-------|
| Version | v0.2.6 |
| License | MIT |
| Purpose | Cross-platform OS keychain access (macOS Keychain, Linux libsecret, Windows Credential Manager) |
| Why not in-house | Each platform requires a different native API (Security.framework, dbus/libsecret, wincred). A portable abstraction would duplicate this library exactly. |

### `github.com/joho/godotenv`

| Field | Value |
|-------|-------|
| Version | v1.5.1 |
| License | MIT |
| Purpose | Parse `.env` files for token fallback in the config layer |
| Why not in-house | Parsing `.env` files has many edge cases (quoting, comments, export prefix). Using a well-tested library avoids subtle bugs in a security-sensitive path. |

### `github.com/danieljoos/wincred`

| Field | Value |
|-------|-------|
| Version | v1.2.2 |
| License | MIT |
| Purpose | Windows Credential Manager backend used by `go-keyring` |
| Compliance | Transitive only; no direct import |

### `github.com/godbus/dbus/v5`

| Field | Value |
|-------|-------|
| Version | v5.1.0 |
| License | BSD-2-Clause |
| Purpose | D-Bus IPC used by `go-keyring` on Linux for libsecret access |
| Compliance | Transitive only; no direct import |

### `al.essio.dev/pkg/shellescape`

| Field | Value |
|-------|-------|
| Version | v1.5.1 |
| License | MIT |
| Purpose | Shell-safe string escaping; pulled in by `go-keyring` |
| Compliance | Transitive only; no direct import |

### `github.com/mattn/go-colorable` / `github.com/mattn/go-isatty`

| Field | Value |
|-------|-------|
| Versions | v0.1.14 / v0.0.20 |
| License | MIT |
| Purpose | Windows stdout colour support and TTY detection used by `fatih/color` |
| Compliance | Transitive only; no direct import |

### `github.com/spf13/pflag`

| Field | Value |
|-------|-------|
| Version | v1.0.9 |
| License | BSD-3-Clause |
| Purpose | POSIX-style flag parsing used internally by cobra |
| Compliance | Transitive only; no direct import |

### `github.com/inconshreveable/mousetrap`

| Field | Value |
|-------|-------|
| Version | v1.1.0 |
| License | Apache-2.0 |
| Purpose | Windows double-click detection used by cobra to warn users who launch a CLI binary from Explorer |
| Compliance | Transitive only; no direct import |

### `golang.org/x/sys`

| Field | Value |
|-------|-------|
| Version | v0.42.0 |
| License | BSD-3-Clause |
| Purpose | Low-level OS syscall helpers used by `go-isatty` and `go-keyring` |
| Compliance | Transitive only; no direct import |

---

## Dev tools (not compiled into the binary)

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Static analysis and lint gate (`make check`) |
| `govulncheck` | Vulnerability scanning of the dependency graph (`make check`) |

---

*Last updated: 2026-03-20*
