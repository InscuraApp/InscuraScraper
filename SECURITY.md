# Security Policy

## Supported Versions

Security fixes are applied to the latest released minor version. Older versions
are supported on a best-effort basis.

| Version | Supported |
| ------- | --------- |
| latest  | ✅        |
| older   | ⚠️ best-effort |

## Reporting a Vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Instead, report privately via one of:

- GitHub Security Advisories: use the "Report a vulnerability" button on the
  repository Security tab (preferred).
- Email the maintainers listed in `CODEOWNERS`.

Please include:

- A description of the issue and its impact.
- Steps to reproduce, a proof-of-concept, or affected endpoints.
- Affected version(s) and configuration.
- Your name/handle for credit (optional).

### Response Timeline

- **Acknowledgement**: within 3 business days.
- **Initial assessment**: within 7 business days.
- **Fix or mitigation**: coordinated with the reporter; typically within 30
  days for confirmed issues, depending on severity.

We follow coordinated disclosure: once a fix is available, we will publish an
advisory and credit the reporter unless they request anonymity.

## Scope

In scope:

- The `inscurascraper-server` HTTP API and its authentication.
- The provider and engine packages.
- Supply-chain issues in pinned dependencies.

Out of scope:

- Vulnerabilities in third-party provider sites scraped by this project.
- Attacks that require an already-compromised environment (stolen tokens,
  local file access).
- Denial of service by sending unbounded upstream traffic.

## Handling Secrets

Never commit API tokens, database credentials, or other secrets. Configure
them via environment variables (see `README.md`). If a secret is exposed,
rotate it and report as a vulnerability.
