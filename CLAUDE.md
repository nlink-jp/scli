# Project Rules

This document defines the fundamental rules and policies for this project.
All contributors (including Claude Code) must follow these rules.

---

## 1. Security First

- Treat security as a first-class concern at every stage of design, implementation, and review.
- Never embed secrets, credentials, or sensitive data in source code.
- Keep dependencies minimal; document the rationale for each third-party library adopted (see Rule 18).
- Integrate security scanning into the local quality gate (see Rule 20) to continuously verify the dependency chain.

## 2. Small and Focused

- Build the smallest unit that satisfies the requirement, then iterate.
- A fix must be scoped to the problem — do not refactor unrelated code in the same change.
- Prefer composition over monolithic structures.

## 3. Separation of Concerns

- Each module, package, or layer must have a single, well-defined responsibility.
- Do not mix I/O, business logic, and presentation in the same unit.
- Define clear boundaries between layers (e.g., transport, domain, persistence) and communicate across them via explicit interfaces.
- Violations of this rule make code harder to test, harder to reason about, and harder to change safely.

## 4. Testable Design

- Design code so that units can be tested independently (dependency injection, clear interfaces).
- Avoid hidden global state; make side effects explicit.
- Keep functions/methods small and single-purpose.

## 5. Implementation and Tests Together

- Write tests alongside the implementation in the same commit or PR.
- Do not merge untested production code.

## 6. Documentation Required

- Every public API, module, and non-trivial design decision must be documented.
- Documentation lives in the `docs/` directory; inline comments supplement but do not replace it.

## 7. Documentation Must Stay in Sync

- When code changes, the corresponding documentation must be updated in the same PR.
- A PR that changes behavior without updating documentation is not complete.

## 8. Test Before Marking Complete

- All tests must pass locally before a feature or fix is considered done.
- Run the full test suite, not just the tests related to the change.

## 9. Commit After Tests Pass

- Only commit (or merge) when all tests are green.
- Commit messages must be descriptive (what changed and why).

## 10. Preserve Recoverability for Large Changes

- Before making a large or risky change, create a dedicated branch so the pre-change state is always reachable.
- Use feature flags or phased rollout when behavioral changes cannot be easily reversed.
- Tag releases before breaking changes are introduced.

## 11. Language Policy for Docs and Comments

- All source code comments and primary documentation are written in **English**.
- A Japanese translation (`docs/ja/`) must be maintained in parallel and kept in sync.

## 12. Communication Language

- All communication between contributors and Claude Code is conducted in **Japanese**.

## 13. Design Before Implementation

- Before writing any production code, step back and review the overall system:
  1. Write a high-level design document (`docs/design/`).
  2. Produce a development plan with phases and milestones.
  3. Get explicit sign-off before starting implementation.
- When integrating with external APIs (e.g., Slack Web API), enumerate **all required OAuth scopes** at design time and cross-check them against every API method that will be called. Discovering a missing scope at runtime is a preventable error.

## 14. Native Code: Go + Make + Cross-Compilation

- Go is the baseline language for native/compiled code.
- Build system: GNU `make` with a `Makefile` at the project root.
- Target platforms: `linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64`. darwin is arm64-only (no Intel).
  - If Windows support is not feasible due to OS-level constraints, `linux` and `darwin` are acceptable.
- Cross-compilation must work from a single host machine (use `GOOS`/`GOARCH` variables).
- Code style: enforced via `gofmt` and `golangci-lint`.
- `golangci-lint` must be installed via `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` to ensure it is built with a Go version ≥ the project's `go` directive. Package-manager installs (e.g., Homebrew) may lag behind and produce a version mismatch error.

## 15. Python: uv Virtual Environments

- Python code must run inside a `uv`-managed virtual environment.
- `pyproject.toml` is the canonical configuration file; `uv.lock` must be committed.
- Code style: enforced via `ruff` (lint + format).

## 16. Sandbox-Aware Build Configuration

- The development environment runs inside a sandbox with restricted filesystem and network access.
- Build scripts must not assume unrestricted outbound network access; vendor or cache dependencies where needed.
- Document any host-level prerequisites in `docs/setup.md`.

## 17. Git and GitHub

- All code is managed with Git; the authoritative remote is GitHub.
- Branch strategy:
  - `main` — protected; direct pushes are prohibited.
  - `feature/<name>` — new features.
  - `fix/<name>` — bug fixes.
  - `docs/<name>` — documentation-only changes.
  - `chore/<name>` — tooling, dependency updates.
- Pull Requests are required to merge into `main`.
- PRs must include a description of what changed and why.

## 18. Dependency Management

- Add third-party dependencies only when genuinely necessary.
- For each dependency added, document in `docs/dependencies.md`:
  - Purpose and why an in-house solution was not preferred.
  - License and any compliance considerations.
- Remove unused dependencies promptly.

## 19. Error and Warning Policy

- Errors must never be silently ignored (no bare `_ = err` in Go, no bare `except: pass` in Python).
- Compiler warnings, linter warnings, and test warnings must not be left unresolved; treat them as errors.
- Use structured logging with consistent severity levels (`DEBUG`, `INFO`, `WARN`, `ERROR`).
- Distinguish between recoverable errors (return/log) and unrecoverable errors (fail fast with a clear message).

## 20. Quality Gates via Local Automation (Git Hooks + Makefile)

- To avoid cloud CI costs, quality gates are enforced locally rather than via a hosted CI service.
- A `make check` target (or equivalent) must:
  1. Lint the code.
  2. Run the full test suite.
  3. Build for all target platforms.
  4. Run security scans (see Rule 21).
- A Git **pre-commit hook** (managed via the repo, e.g., under `scripts/hooks/`) runs `make check` before every commit; commits are rejected if any step fails.
- A Git **pre-push hook** performs the same checks before pushing to the remote.
- Hook installation is documented in `docs/setup.md` and can be automated with `make setup`.

## 21. Security Scanning

- Go: run `govulncheck ./...` as part of `make check`.
- Python: run `pip-audit` (or `uv run pip-audit`) as part of `make check`.
- Address any findings before merging; if a finding is accepted as low-risk, document the rationale.

## 22. Versioning and Changelog

- Follow [Semantic Versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`).
- Maintain a `CHANGELOG.md` updated with every release.
- Tag releases in Git (`v1.2.3`) before distributing artifacts.

### Release artifact process

1. Ensure the working tree is clean (`git status` shows no modifications) so that `git describe --tags --dirty` produces a clean version string (e.g., `v1.2.3`).
2. Run `make build-all` to produce binaries under `dist/` with the correct embedded version.
3. Zip each binary: `zip scli-<os>-<arch>-<version>.zip scli-<os>-<arch>`.
4. Create the GitHub Release with `gh release create <tag> dist/*.zip --title "<tag>" --notes "..."`.
5. Restore the SSH remote URL if it was temporarily switched to HTTPS for pushing.

---

*Primary language for this document: English. Japanese translation: `docs/ja/RULES.md`*
