# AGENTS.md

Technical operating rules for this Go repository template.

## 1. Repository Scope

- Public pure-Go workspace template with native `cmd/xtask` automation and zero external runtime scripts.
- Update template placeholders (`CHANGE_ME`) in `go.mod`, `LICENSE`, `NOTICE`, `README.md`, and `.github/CODEOWNERS` before production use.

## 2. Toolchain & Validation

- Enable hooks locally with `git config core.hooksPath .githooks` and run `go run ./cmd/xtask validate` (`TOTAL PASS=8 FAIL=0`) before opening or updating any pull request.
- Do not run `gofmt` or automated code formatters in hooks, CI, or pre-commit steps.

## 3. Code Layout & Architecture

- Organize domain logic under `internal/modules/<domain>/` as a six-file layered modular layout (`entity.go`, `dto.go`, `repository.go`, `service.go`, `controller.go`, `module.go`).
- Enforce unidirectional imports: `Controller` calls `Service` and `dto` (never `Repository`); `entity` and `dto` must not reference `service`, `repository`, or `controller`.
- Keep every source and documentation file at or below **500 lines**.

## 4. Commit, Dependency & License Policy

- Set Git Author and Committer identically to your maintainer handle and GitHub noreply email (`<id>+<handle>@users.noreply.github.com`).
- Format commits as `type(scope): description` (3 to 6 lowercase words, no trailing period, max 100 chars), a single blank line, and 3+ consecutive `- **Label**: text` bullets (3 to 10 words each) without trailers.
- External dependencies must pass the 3-day publish cooldown (`.github/dependabot.yml` `cooldown: default-days: 3`, no `replace` directives, no `@latest`); run `go run ./cmd/xtask notices` when `go.mod` changes.
- Code and documentation are released under the Apache License 2.0 (`Copyright (c) 2026 CHANGE_ME`).
