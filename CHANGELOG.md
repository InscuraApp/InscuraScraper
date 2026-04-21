# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1] - 2026-04-21

### Added
- Open-source governance: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  `SECURITY.md`, `CODEOWNERS`, `CHANGELOG.md`.
- GitHub issue and pull request templates under `.github/`.
- GitHub Actions workflow for lint, test, and Docker build.
- Dependabot configuration for Go modules, GitHub Actions, and Docker.
- `/healthz` and `/readyz` health-check endpoints.
- Godoc coverage for public provider interfaces.

### Changed
- `.gitignore` now excludes local databases, IDE, OS, and coverage artifacts.

### Removed
- Local PostgreSQL `db/` directory is no longer tracked in version control.

[Unreleased]: https://github.com/InscuraApp/InscuraScraper/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/InscuraApp/InscuraScraper/releases/tag/v0.0.1
