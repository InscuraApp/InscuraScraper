# Contributing to InscuraScraper

Thank you for your interest in contributing! This document explains how to
set up your environment, the workflow we expect, and the conventions the
project follows.

## Ways to Contribute

- Report bugs via GitHub Issues (use the bug template).
- Propose features or provider integrations (use the feature template).
- Submit pull requests — fixes, new providers, documentation, or tests.
- Improve docs and examples.

## Development Setup

Prerequisites:

- Go 1.25 or newer.
- `make`.
- Optional: Docker / Docker Compose for running PostgreSQL.
- Optional: `golangci-lint` v2 for local linting.

Clone and build:

```sh
git clone https://github.com/InscuraApp/InscuraScraper.git
cd InscuraScraper
make            # development build → build/inscurascraper-server
```

Run the server (in-memory SQLite, no auth):

```sh
./build/inscurascraper-server
```

Run with PostgreSQL via Docker Compose:

```sh
docker compose up
```

## Workflow

1. Fork the repository and create a topic branch from `main`
   (e.g. `feat/new-provider`, `fix/tmdb-timeout`).
2. Make focused commits with descriptive messages.
3. Run `make lint` and `go test ./...` locally before pushing.
4. Open a pull request against `main`. Fill in the PR template.
5. Address review feedback; squash noise commits on request.

We follow [Conventional Commits](https://www.conventionalcommits.org/) for
commit messages and PR titles:

```
feat(provider/tmdb): add review endpoint
fix(engine): avoid duplicate search results
docs(readme): document IS_ prefix env vars
```

## Code Style

- Formatting: `gofumpt` + `gci` (see `.golangci.yaml`). Most editors can run
  these on save.
- Import grouping: stdlib → third-party → `inscurascraper/...`.
- Lint: `golangci-lint run ./...` must pass.
- Tests: co-located `_test.go`; use table-driven tests where sensible.
- Exported identifiers need a godoc comment that starts with the identifier
  name.

## Adding a Provider

See the **Provider Development Guide** in `CLAUDE.md` for the full checklist.
In summary:

1. Create `provider/<name>/` and embed `*scraper.Scraper`.
2. Implement `MovieProvider` and/or `ActorProvider` from
   `provider/provider.go`.
3. Register in `init()` via `provider.Register(Name, New)`.
4. Blank-import in `engine/register.go`.
5. Add tests and update `docs/` / `README.md` as needed.

## Tests

Run the full suite:

```sh
go test ./...
```

Run a single package or test:

```sh
go test ./provider/tmdb/...
go test ./engine/dbengine/ -run TestXxx
```

Integration tests that require network or credentials should be guarded by
build tags or `testing.Short()` so they do not block CI by default.

## Reporting Security Issues

Do not open public issues for security vulnerabilities. Follow the process in
[`SECURITY.md`](SECURITY.md).

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
Participation is subject to its terms.

## License

By contributing, you agree that your contributions will be licensed under the
project's [Apache-2.0 License](LICENSE).
