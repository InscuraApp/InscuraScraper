# InscuraScraper

**English** | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://golang.org/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?logo=github-actions)](.github/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/ghcr.io-inscuraapp%2Finscurascraper-2496ED?logo=docker)](https://github.com/orgs/InscuraApp/packages/container/package/inscurascraper)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

**InscuraScraper** is a metadata-scraping SDK and HTTP service written in Go. It pulls movie and actor metadata from sources such as TMDB, TVDB, TVmaze, AniList, and Fanart.tv through a pluggable provider architecture, exposes a unified RESTful API, and uses SQLite or PostgreSQL for local caching.

> Forked and refactored with the original author's permission.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
  - [Binary](#binary)
  - [Docker](#docker)
  - [Docker Compose](#docker-compose)
- [Configuration](#configuration)
  - [Server Options](#server-options)
  - [Provider Configuration (Environment Variables)](#provider-configuration-environment-variables)
- [API Reference](#api-reference)
  - [Authentication](#authentication)
  - [Common Response Shape](#common-response-shape)
  - [Optional Request Headers](#optional-request-headers)
  - [Endpoint Overview](#endpoint-overview)
  - [Endpoint Details](#endpoint-details)
- [Data Models](#data-models)
- [Development](#development)
- [Contributing / Security / License](#contributing--security--license)

## Features

- 🔌 **Pluggable provider architecture** — TMDB, TVDB, TVmaze, AniList, and Fanart.tv built in; new sources only need to implement the interface and register
- 🚀 **RESTful API** — Gin-driven, unified endpoints for search / info / reviews / proxy configuration
- 🗄️ **Dual database support** — in-memory SQLite by default (zero config); switch to PostgreSQL for production
- ⚡ **Local cache** — serve from cache first, fall back to upstream to conserve quota
- 🌐 **Per-request overrides** — swap proxy, API key, or language via request headers without restart
- 💊 **Observability** — built-in `/healthz` and `/readyz` probes
- 🐳 **Cross-platform** — Linux / macOS / Windows / BSD, Dockerfile and Docker Compose included

## Quick Start

> 💡 **Ready to go**: the Docker image is published on GHCR — run `docker pull ghcr.io/inscuraapp/inscurascraper:latest` to get started. See [Docker](#docker).

### Binary

Prerequisites: Go 1.25+, `make`.

```sh
git clone https://github.com/InscuraApp/InscuraScraper.git
cd InscuraScraper
make                                  # output: build/inscurascraper-server

./build/inscurascraper-server         # listens on :8080 with in-memory SQLite
```

Verify:

```sh
curl -s http://localhost:8080/healthz
# {"status":"ok"}

curl -s http://localhost:8080/v1/providers | jq
```

### Docker

Images are published to **GitHub Container Registry**: `ghcr.io/inscuraapp/inscurascraper`.

**Available tags:**

| Tag | Meaning |
|-----|---------|
| `latest` | Latest stable release |
| `vX.Y.Z` | Specific version (recommended for production, e.g. `v0.0.1`) |
| `X.Y` | Pin to a minor line (e.g. `0.0`) and auto-receive patch updates |

**Supported architectures:** `linux/amd64`, `linux/arm64`

#### Pull and run

```sh
# Latest version, in-memory SQLite, no auth
docker run --rm -p 8080:8080 \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

#### With persistent SQLite file

Mount the database file onto the host to survive container rebuilds:

```sh
mkdir -p ./data

docker run -d --name inscurascraper -p 8080:8080 \
  -v $PWD/data:/data \
  -e TOKEN=change-me \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest \
  -dsn "/data/inscurascraper.db" -db-auto-migrate
```

#### Build locally (optional)

If you prefer to build from source instead of pulling the prebuilt image:

```sh
docker build -t inscurascraper:local .
docker run --rm -p 8080:8080 inscurascraper:local
```

### Docker Compose

The repo ships `docker-compose.yaml` to bring up InscuraScraper + PostgreSQL in one command.

> **Note**: the current `docker-compose.yaml` defaults to the local image `inscurascraper-server:latest`. To use the GHCR image directly, change `image:` to `ghcr.io/inscuraapp/inscurascraper:latest` — no `docker build` required.

```sh
# Option 1: use the GHCR image (recommended)
#   edit docker-compose.yaml: change image: inscurascraper-server:latest
#   to image: ghcr.io/inscuraapp/inscurascraper:latest

# Option 2: build locally (requires source)
docker build -t inscurascraper-server:latest .

# Start
docker compose up -d

# Tail logs
docker compose logs -f inscurascraper
```

The first run auto-creates tables (`-db-auto-migrate`). Inject your API tokens via the `environment` block in `docker-compose.yaml`, or via a `.env` file:

```env
IS_PROVIDER_TMDB__API_TOKEN=xxxxx
IS_PROVIDER_FANARTTV__API_KEY=xxxxx
IS_PROVIDER_TVDB__API_KEY=xxxxx
IS_PROVIDER_TVMAZE__API_KEY=xxxxx
```

> **Note**: `docker-compose.yaml` mounts the PostgreSQL data volume at `./db` inside the project; that directory is already in `.gitignore` — do not commit it.

## Configuration

### Server Options

All options can be set via **command-line flags** or **uppercase environment variables of the same name** (parsed by `peterbourgon/ff`).

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-bind` | `BIND` | `""` | Bind address (empty = listen on all interfaces) |
| `-port` | `PORT` | `8080` | HTTP port |
| `-token` | `TOKEN` | `""` | API auth token; empty disables authentication |
| `-dsn` | `DSN` | `""` | Database DSN; empty = in-memory SQLite |
| `-request-timeout` | `REQUEST_TIMEOUT` | `1m` | Per-upstream-request timeout |
| `-db-auto-migrate` | `DB_AUTO_MIGRATE` | `false` | Auto-create tables on startup (forced on for SQLite) |
| `-db-prepared-stmt` | `DB_PREPARED_STMT` | `false` | Enable prepared statements |
| `-db-max-idle-conns` | `DB_MAX_IDLE_CONNS` | `0` | Max idle DB connections |
| `-db-max-open-conns` | `DB_MAX_OPEN_CONNS` | `0` | Max open DB connections |
| `-version` | `VERSION` | - | Print version and exit |

DSN examples:

```sh
# SQLite file
-dsn "/data/inscurascraper.db"

# PostgreSQL TCP
-dsn "postgres://user:pass@host:5432/inscurascraper?sslmode=disable"

# PostgreSQL Unix socket (see docker-compose.yaml)
-dsn "postgres://user:pass@/inscurascraper?host=/var/run/postgresql"
```

### Provider Configuration (Environment Variables)

Per-provider API keys, proxies, and priorities are injected via prefixed environment variables:

```sh
# Applies to both actor and movie providers
IS_PROVIDER_{NAME}__{KEY}=value

# Actor provider only
IS_ACTOR_PROVIDER_{NAME}__{KEY}=value

# Movie provider only
IS_MOVIE_PROVIDER_{NAME}__{KEY}=value
```

Common `{KEY}`s:

| Key | Description |
|-----|-------------|
| `API_TOKEN` / `API_KEY` | Upstream API credential |
| `PRIORITY` | Match priority (higher wins) |
| `PROXY` | HTTP/SOCKS5 proxy URL |
| `TIMEOUT` | Request timeout (Go duration, e.g. `30s`) |

Example:

```sh
export IS_PROVIDER_TMDB__API_TOKEN=eyJhbGciOi...
export IS_PROVIDER_TMDB__PRIORITY=10
export IS_PROVIDER_TMDB__PROXY=http://127.0.0.1:7890
```

## API Reference

### Authentication

InscuraScraper protects **private endpoints** (paths marked ✅ in the [Endpoint Overview](#endpoint-overview)) with a simple **Bearer Token** scheme. Public endpoints (`/`, `/healthz`, `/readyz`, `/v1/modules`, `/v1/providers`, `/?redirect=...`) are unaffected.

#### Enable authentication

Configure the token via a **command-line flag** or an **environment variable** (flag takes precedence):

```sh
# Option A: command-line flag
./build/inscurascraper-server -token "my-secret-token"

# Option B: environment variable
export TOKEN="my-secret-token"
./build/inscurascraper-server
```

**When the token is empty (`-token ""`), authentication is fully disabled** and every endpoint becomes public — fine for local development or internal networks, but you must set it explicitly in production.

Docker:

```sh
docker run -d -p 8080:8080 \
  -e TOKEN=my-secret-token \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

Docker Compose — add to the `environment` block in `docker-compose.yaml`:

```yaml
services:
  inscurascraper:
    environment:
      TOKEN: my-secret-token
```

Or load via a `.env` file at the repo root:

```env
TOKEN=my-secret-token
```

> 💡 Use a sufficiently long random string (e.g. `openssl rand -hex 32`) and inject it via a secret manager. Do not commit it to the repo or bake it into images.

#### Calling private endpoints

Attach the token as a request header. **The format must be exactly `Bearer <token>`** (case-sensitive):

```sh
curl -H "Authorization: Bearer my-secret-token" \
  "http://localhost:8080/v1/movies/search?q=Inception"
```

Validation failures always return:

```
HTTP/1.1 401 Unauthorized
```

```json
{ "error": { "code": 401, "message": "unauthorized" } }
```

Common causes:

- Missing `Authorization` header
- Prefix is not `Bearer` (case-sensitive — `bearer`, `Token`, etc. are rejected)
- Token value does not match the server-side configuration

#### Rotating or revoking tokens

The current implementation is a **single static token**. Rotating requires restarting the process to pick up the new value; for multi-token management or dynamic revocation, extend `route/auth.TokenStore` in code.

### Common Response Shape

Every endpoint returns:

```json
{
  "data": { },
  "error": { "code": 400, "message": "..." }
}
```

- Success: only `data` is populated
- Failure: only `error` is populated; the HTTP status code matches `error.code`

### Optional Request Headers

You can override provider behavior per request without restarting:

| Header | Description |
|--------|-------------|
| `X-Is-Proxy` | Proxy URL applied to all providers for this request |
| `X-Is-Api-Key-{PROVIDER}` | Override API key for the named provider (case-insensitive) |
| `X-Is-Language` | Response language (BCP 47 tag, e.g. `zh-CN`, `en-US`) |

Example:

```sh
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: en-US" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     "http://localhost:8080/v1/movies/search?q=Inception"
```

### Endpoint Overview

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/` | ❌ | Service info |
| GET | `/healthz` | ❌ | Liveness probe |
| GET | `/readyz` | ❌ | Readiness probe (checks database) |
| GET | `/v1/modules` | ❌ | Build dependency list |
| GET | `/v1/providers` | ❌ | Registered providers |
| GET | `/v1/db/version` | ✅ | Database version |
| GET | `/v1/config/proxy` | ✅ | Current provider proxy config |
| GET | `/v1/movies/search` | ✅ | Search movies |
| GET | `/v1/movies/:provider/:id` | ✅ | Movie details |
| GET | `/v1/actors/search` | ✅ | Search actors |
| GET | `/v1/actors/:provider/:id` | ✅ | Actor details |
| GET | `/v1/reviews/:provider/:id` | ✅ | Movie reviews |
| GET | `/?redirect=:provider:id` | ❌ | Redirect to the upstream page for a given provider/ID |

### Endpoint Details

#### `GET /`

```sh
curl -s http://localhost:8080/
```

```json
{
  "data": {
    "app": "inscurascraper",
    "version": "v0.0.1 (abc1234)"
  }
}
```

#### `GET /healthz` / `GET /readyz`

```sh
curl -s http://localhost:8080/healthz
# {"status":"ok"}

curl -s http://localhost:8080/readyz
# {"status":"ready"}
# If the database is unreachable: HTTP 503 {"status":"not_ready","error":"..."}
```

#### `GET /v1/providers`

```sh
curl -s http://localhost:8080/v1/providers | jq
```

```json
{
  "data": {
    "actor_providers": {
      "TMDB": "https://www.themoviedb.org",
      "TVDB": "https://thetvdb.com"
    },
    "movie_providers": {
      "TMDB":   "https://www.themoviedb.org",
      "TVmaze": "https://www.tvmaze.com"
    }
  }
}
```

#### `GET /v1/movies/search`

Query parameters:

| Param | Required | Description |
|-------|----------|-------------|
| `q` | ✅ | Keyword; if an http(s) URL is supplied, the provider and ID are parsed and a detail fetch is performed |
| `provider` | ❌ | Restrict to a single provider (case-insensitive); omit to aggregate across all providers |
| `fallback` | ❌ | When upstream returns no results, whether to fall back to the local DB cache (default `true`) |

```sh
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/search?q=Inception&provider=TMDB" | jq
```

```json
{
  "data": [
    {
      "id": "27205",
      "number": "tt1375666",
      "title": "Inception",
      "provider": "TMDB",
      "homepage": "https://www.themoviedb.org/movie/27205",
      "thumb_url": "https://image.tmdb.org/t/p/w300/...jpg",
      "cover_url": "https://image.tmdb.org/t/p/original/...jpg",
      "score": 8.4,
      "actors": ["Leonardo DiCaprio", "Joseph Gordon-Levitt"],
      "release_date": "2010-07-15"
    }
  ]
}
```

#### `GET /v1/movies/:provider/:id`

Query parameters:

| Param | Description |
|-------|-------------|
| `lazy` | `true` (default) = prefer cache; `false` = force a fresh upstream fetch |

```sh
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/TMDB/27205?lazy=false" | jq
```

```json
{
  "data": {
    "id": "27205",
    "number": "tt1375666",
    "title": "Inception",
    "summary": "Cobb, a skilled thief who commits corporate espionage...",
    "provider": "TMDB",
    "homepage": "https://www.themoviedb.org/movie/27205",
    "director": "Christopher Nolan",
    "actors": ["Leonardo DiCaprio", "Joseph Gordon-Levitt", "Elliot Page"],
    "thumb_url": "https://image.tmdb.org/t/p/w300/...jpg",
    "big_thumb_url": "https://image.tmdb.org/t/p/w780/...jpg",
    "cover_url": "https://image.tmdb.org/t/p/original/...jpg",
    "big_cover_url": "https://image.tmdb.org/t/p/original/...jpg",
    "preview_video_url": "",
    "preview_video_hls_url": "",
    "preview_images": [],
    "maker": "Legendary Pictures",
    "label": "",
    "series": "",
    "genres": ["Action", "Science Fiction", "Adventure"],
    "score": 8.4,
    "runtime": 148,
    "release_date": "2010-07-15"
  }
}
```

#### `GET /v1/actors/search` / `GET /v1/actors/:provider/:id`

Parameters mirror the movie endpoints. Sample actor payload:

```json
{
  "data": {
    "id": "6193",
    "name": "Leonardo DiCaprio",
    "provider": "TMDB",
    "homepage": "https://www.themoviedb.org/person/6193",
    "summary": "Leonardo Wilhelm DiCaprio is an American actor...",
    "aliases": ["Leo"],
    "images": [
      "https://image.tmdb.org/t/p/original/...jpg"
    ],
    "nationality": "US",
    "height": 183,
    "birthday": "1974-11-11",
    "debut_date": "1991-01-01"
  }
}
```

#### `GET /v1/reviews/:provider/:id`

Query parameters:

| Param | Description |
|-------|-------------|
| `homepage` | Optional; scrape reviews directly from a URL |
| `lazy` | Same as above |

```sh
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/reviews/TMDB/27205" | jq
```

```json
{
  "data": [
    {
      "title": "A modern classic",
      "author": "cinemaphile",
      "comment": "Nolan at his peak...",
      "score": 9.0,
      "date": "2020-06-01"
    }
  ]
}
```

> Only providers implementing the `MovieReviewer` interface support this endpoint; others return 400.

#### `GET /v1/db/version`

```json
{ "data": { "version": "PostgreSQL 15.6 on x86_64-pc-linux-musl ..." } }
```

#### `GET /v1/config/proxy`

Returns each provider's persisted proxy setting (injected via environment variables; read-only at runtime).

```json
{
  "data": {
    "TMDB":   "http://127.0.0.1:7890",
    "TVDB":   ""
  }
}
```

#### `GET /?redirect=TMDB:27205`

Issues a `302` redirect to the upstream homepage for the given movie/actor.

### Error Responses

```json
{
  "error": {
    "code": 404,
    "message": "info not found"
  }
}
```

Common status codes:

| HTTP | Meaning |
|------|---------|
| 400 | Bad parameter / malformed ID or URL |
| 401 | Missing or invalid token |
| 404 | Resource or provider not found |
| 500 | Upstream scrape failure / database error |
| 503 | `/readyz`: database unreachable |

## Data Models

See [`model/movie.go`](model/movie.go) and [`model/actor.go`](model/actor.go) for the full field definitions. Summary:

- `MovieInfo`: `id, number, title, summary, provider, homepage, director, actors[], thumb_url, big_thumb_url, cover_url, big_cover_url, preview_video_url, preview_video_hls_url, preview_images[], maker, label, series, genres[], score, runtime, release_date`
- `MovieSearchResult`: a lightweight subset of `MovieInfo`
- `ActorInfo`: `id, name, provider, homepage, summary, aliases[], images[], birthday, blood_type, cup_size, measurements, height, nationality, debut_date`
- `MovieReviewDetail`: `title, author, comment, score, date`

## Development

### Build / Test / Lint

```sh
make              # Dev build
make server       # Production build
make lint         # golangci-lint
go test ./...     # Full test suite
```

Cross-compile:

```sh
make darwin-arm64 linux-amd64 windows-amd64
make releases          # Emit zips for all architectures under build/
```

### Developing a New Provider

See the **Provider Development Guide** in [CLAUDE.md](CLAUDE.md) and [CONTRIBUTING.md](CONTRIBUTING.md) for the full walkthrough. In brief:

1. Create `provider/<name>/` and embed `*scraper.Scraper`
2. Implement `provider.MovieProvider` and/or `ActorProvider`
3. Call `provider.Register(Name, New)` in `init()`
4. Add a blank import in `engine/register.go`

## Contributing / Security / License

- [Contributing guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security policy](SECURITY.md) (do not file public issues)
- [Changelog](CHANGELOG.md)
- License: [Apache 2.0](LICENSE)

## Acknowledgements

| Library | Description |
|---------|-------------|
| [gocolly/colly](https://github.com/gocolly/colly) | Elegant scraper and crawler framework for Go |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP web framework |
| [gorm.io/gorm](https://gorm.io/) | ORM for Go |
| [robertkrimen/otto](https://github.com/robertkrimen/otto) | Pure-Go JavaScript interpreter |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | CGo-free SQLite port |
| [antchfx/xpath](https://github.com/antchfx/xpath) | XPath for HTML / XML / JSON |
| [peterbourgon/ff](https://github.com/peterbourgon/ff) | Flags / env unified parsing |
