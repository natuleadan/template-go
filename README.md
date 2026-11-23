# Project Template (CHANGE ME)

Production-ready pure-Go repository template featuring a native `cmd/xtask` validation suite, layered modular domain architecture, strict Git hygiene, 3-day dependency cooldown enforcement, and automated GitHub Actions pipelines.

## Key Features

- **100% Native Go Tooling**: Powered by `go run ./cmd/xtask` and standard library checks with zero Node.js or external script dependencies.
- **Layered Modular Architecture**: Reference domain module (`internal/modules/math/`) separating `entity`, `dto`, `repository`, `service`, `controller`, and `module` layers with automated boundary tests.
- **Eight Standardized Validation Gates**: `lint`, `typecheck`, `unit tests`, `integration and architecture tests`, `cooldown`, `hygiene`, `notices`, and `release`.
- **Supply-Chain Hardening**: 3-day dependency cooldown policy (`.github/dependabot.yml` `default-days: 3`, no `replace` or `@latest` directives), CodeQL static analysis, and automated SBOM/dependency inventory artifacts.
- **Zero Formatter Drift**: Automated formatters (`gofmt`, `prettier`) are intentionally excluded from hooks and CI so diffs stay focused on logic.

## Repository Layout

```text
.
├── .githooks/                   # Versioned commit-msg, pre-commit, and pre-push hooks
├── .github/
│   ├── CODEOWNERS               # Code ownership rules (CHANGE ME)
│   ├── dependabot.yml           # gomod and GitHub Actions updates with 3-day cooldown
│   └── workflows/               # validate.yml, nightly-tag.yml, security.yml
├── cmd/xtask/                   # Native Go automation CLI (go run ./cmd/xtask <cmd>)
├── internal/
│   ├── core/                    # HTTP mux (/health, /greet, /math/add) and tests
│   └── modules/math/            # Layered domain module (entity, dto, repository, service, controller, module)
├── ARCHITECTURE.md              # Modular architecture and dependency boundary guide
├── AGENTS.md                    # Autonomous agent operating rules
├── CONTRIBUTING.md              # Contributor workflow and commit grammar
├── SECURITY.md                  # Supply-chain controls and vulnerability reporting
├── LICENSE                      # Apache-2.0 License (Copyright (c) 2026 CHANGE_ME)
└── ThirdPartyNotices.txt        # Generated dependency notice inventory
```

## Quickstart

Enable repository hooks and run the full 8-gate validation suite:

```sh
git config core.hooksPath .githooks
go run ./cmd/xtask validate
```

Other `xtask` commands:

```sh
go run ./cmd/xtask notices          # Regenerate ThirdPartyNotices.txt from go list
go run ./cmd/xtask release --check  # Verify VERSION matches CHANGELOG.md
```

## Validation Gates

| Gate | Implementation | Purpose |
| :--- | :--- | :--- |
| `lint` | `go vet ./...` + AST checks | Static lints and forbidden calls |
| `typecheck` | `go build ./...` | Full package compilation check |
| `unit tests` | `go test -run TestUnit ./internal/...` | Fast unit test execution |
| `integration and architecture tests` | `go test -run 'TestIntegration\|TestArchitecture' ./internal/...` | HTTP endpoints and layer boundaries |
| `cooldown` | `cmd/xtask/main.go` + `.github/dependabot.yml` | 3-day dependency cooldown enforcement |
| `hygiene` | `cmd/xtask/hygiene.go` | Anti-leak, required docs, path checks |
| `notices` | `cmd/xtask/notices.go` | License inventory parity |
| `release` | `cmd/xtask/release.go` | SemVer and `CHANGELOG.md` consistency |

## Branching & Release Workflow

- **`dev`**: Active integration branch protected against deletion. Every push updates the rolling `dev` tag, `dev-<short_sha>` tag, and `dev` pre-release snapshot.
- **`main`**: Protected release branch. Accepts rebase-only pull requests from `dev` after all status checks pass. Merging triggers automatic `vX.Y.Z` tagging, GitHub Release creation, and `dev` fast-forward sync.

## Template Customization (CHANGE ME)

When bootstrapping a new repository from this template, update the following placeholders:

- **`README.md`**: Replace `# Project Template (CHANGE ME)` with your project title and description.
- **`LICENSE` and `NOTICE`**: Replace `Copyright (c) 2026 CHANGE_ME` with your copyright holder.
- **`.github/CODEOWNERS`**: Uncomment and set your GitHub handle or team (`* @your-handle`, CHANGE ME).
- **Go Module Path**: Replace `module example.com/change-me` in `go.mod` and internal Go imports (`CHANGE ME`).
- **Git Identity**: Configure `git config user.name` and a GitHub noreply address (`<id>+<handle>@users.noreply.github.com`) locally before committing.

## License

Licensed under the [Apache-2.0 License](LICENSE). Copyright (c) 2026 CHANGE_ME.
