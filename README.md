# InscuraScraper

**English** | [简体中文](README.zh-CN.md)

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://golang.org/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?logo=github-actions)](.github/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/ghcr.io-inscuraapp%2Finscurascraper-2496ED?logo=docker)](https://github.com/orgs/InscuraApp/packages/container/package/inscurascraper)

**InscuraScraper** is a metadata-scraping SDK and HTTP service written in Go. It pulls movie and actor metadata from sources such as TMDB, TVDB, TVmaze, AniList, Fanart.tv, and Trakt through a pluggable provider architecture, exposes a unified RESTful API, and uses SQLite or PostgreSQL for local caching.

> Forked and refactored with the original author's permission.

## Table of Contents

- [Running](#running)
  - [Docker](#docker)
  - [Docker Compose](#docker-compose)
  - [Binary](#binary)
- [Configuration](#configuration)
  - [Server Options](#server-options)
  - [Provider Variables](#provider-variables)
  - [Per-Request Override Headers](#per-request-override-headers)
  - [Query Parameters](#query-parameters)
- [Supported Languages](#supported-languages)
- [API Reference](#api-reference)
  - [Authentication](#authentication)
  - [Response Format](#response-format)
  - [Endpoint Overview](#endpoint-overview)
  - [Endpoint Details](#endpoint-details)
  - [Error Responses](#error-responses)
- [Integrations](#integrations)
- [Data Models](#data-models)
- [Development](#development)
- [License & Acknowledgements](#license--acknowledgements)

---

## Running

### Docker

Images are published to **GitHub Container Registry**: `ghcr.io/inscuraapp/inscurascraper`

**Available tags:**

| Tag | Meaning |
|-----|---------|
| `latest` | Latest stable release |
| `vX.Y.Z` | Specific version (recommended for production, e.g. `v0.0.1`) |
| `X.Y` | Pin to a minor line (e.g. `0.0`) and auto-receive patch updates |

**Supported architectures:** `linux/amd64`, `linux/arm64`

**Quickest start — in-memory SQLite, no auth:**

```sh
docker run --rm -p 8080:8080 \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

**With persistent SQLite (survives container rebuilds):**

```sh
mkdir -p ./data
docker run -d --name inscurascraper -p 8080:8080 \
  -v $PWD/data:/data \
  -e TOKEN=change-me \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest \
  -dsn "/data/inscurascraper.db" -db-auto-migrate
```

**Build locally (optional):**

```sh
docker build -t inscurascraper:local .
docker run --rm -p 8080:8080 inscurascraper:local
```

**Verify:**

```sh
curl -s http://localhost:8080/healthz   # {"status":"ok"}
curl -s http://localhost:8080/v1/providers | jq
```

### Docker Compose

The repo ships `docker-compose.yaml` that starts InscuraScraper + PostgreSQL with a single command.

> **Note**: `docker-compose.yaml` defaults to the local image `inscurascraper-server:latest`. To use the GHCR image directly, change `image:` to `ghcr.io/inscuraapp/inscurascraper:latest`.

```sh
# Option A: use GHCR image (recommended — no build step)
#   Edit docker-compose.yaml: set image to ghcr.io/inscuraapp/inscurascraper:latest

# Option B: build locally
docker build -t inscurascraper-server:latest .

docker compose up -d
docker compose logs -f inscurascraper
```

Inject API tokens via environment or a `.env` file at the repo root:

```env
TOKEN=change-me
IS_PROVIDER_TMDB__API_TOKEN=xxxxx
IS_PROVIDER_FANARTTV__API_KEY=xxxxx
IS_PROVIDER_TVDB__API_KEY=xxxxx
IS_PROVIDER_TRAKT__CLIENT_ID=xxxxx
```

> **Note**: PostgreSQL data is mounted to `./db` — already in `.gitignore`. Do not commit it.

### Binary

Prerequisites: Go 1.25+, `make`.

```sh
git clone https://github.com/InscuraApp/InscuraScraper.git
cd InscuraScraper
make                              # output: build/inscurascraper-server
./build/inscurascraper-server     # listens on :8080, in-memory SQLite
```

---

## Configuration

### Server Options

All options can be set via **command-line flags** or **uppercase environment variables of the same name** (parsed by `peterbourgon/ff`). Flags take precedence.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-bind` | `BIND` | `""` | Bind address (empty = all interfaces) |
| `-port` | `PORT` | `8080` | HTTP port |
| `-token` | `TOKEN` | `""` | API auth token; empty disables authentication |
| `-dsn` | `DSN` | `""` | Database DSN; empty = in-memory SQLite |
| `-request-timeout` | `REQUEST_TIMEOUT` | `1m` | Per-upstream-request timeout |
| `-db-auto-migrate` | `DB_AUTO_MIGRATE` | `false` | Auto-create tables on startup (forced on for SQLite) |
| `-db-prepared-stmt` | `DB_PREPARED_STMT` | `false` | Enable prepared statements |
| `-db-max-idle-conns` | `DB_MAX_IDLE_CONNS` | `0` | Max idle DB connections |
| `-db-max-open-conns` | `DB_MAX_OPEN_CONNS` | `0` | Max open DB connections |
| `-version` | — | — | Print version and exit |

**DSN examples:**

```sh
-dsn "/data/inscurascraper.db"                                         # SQLite file
-dsn "postgres://user:pass@host:5432/inscurascraper?sslmode=disable"  # PostgreSQL TCP
-dsn "postgres://user:pass@/inscurascraper?host=/var/run/postgresql"   # PostgreSQL Unix socket
```

### Provider Variables

Per-provider API keys, proxies, and priorities are injected at startup via prefixed environment variables. These are **global settings** — see [Per-Request Override Headers](#per-request-override-headers) for per-call overrides.

**Naming pattern:**

```
IS_PROVIDER_{NAME}__{KEY}=value           # applies to both actor and movie sub-providers
IS_ACTOR_PROVIDER_{NAME}__{KEY}=value     # actor sub-provider only
IS_MOVIE_PROVIDER_{NAME}__{KEY}=value     # movie sub-provider only
```

`{NAME}` is the provider name in **UPPERCASE** (e.g. `TMDB`, `TVDB`, `TRAKT`).

**Common keys (any provider):**

| Key | Type | Description |
|-----|------|-------------|
| `PRIORITY` | float | Match priority — higher wins when multiple providers return results |
| `PROXY` | string | HTTP or SOCKS5 proxy, e.g. `http://127.0.0.1:7890` or `socks5://127.0.0.1:1080` |
| `TIMEOUT` | duration | Per-request timeout in Go duration format, e.g. `30s`, `2m` |

**Provider credential keys:**

| Provider | Env Var | Required | Where to get |
|----------|---------|----------|--------------|
| **TMDB** | `IS_PROVIDER_TMDB__API_TOKEN` | Yes | [themoviedb.org/settings/api](https://www.themoviedb.org/settings/api) — use the **Bearer Token (v4 auth)** |
| **TVDB** | `IS_PROVIDER_TVDB__API_KEY` | Yes | [thetvdb.com/api-information](https://thetvdb.com/api-information) |
| **Fanart.tv** | `IS_PROVIDER_FANARTTV__API_KEY` | Yes | [fanart.tv/get-an-api-key](https://fanart.tv/get-an-api-key/) |
| **Trakt** | `IS_PROVIDER_TRAKT__CLIENT_ID` | Yes | [trakt.tv/oauth/applications](https://trakt.tv/oauth/applications) — register an app, copy **Client ID** |
| **AniList** | *(none)* | — | Public API, no key required |
| **TVmaze** | *(none)* | — | Public API, no key required |
| **Bangumi** | *(none)* | — | Public API, no key required |

**Full example:**

```sh
export IS_PROVIDER_TMDB__API_TOKEN=eyJhbGciOiJSUzI1NiJ9...
export IS_PROVIDER_TVDB__API_KEY=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
export IS_PROVIDER_FANARTTV__API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export IS_PROVIDER_TRAKT__CLIENT_ID=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

export IS_PROVIDER_TMDB__PRIORITY=1000
export IS_PROVIDER_TMDB__PROXY=http://127.0.0.1:7890
export IS_PROVIDER_TMDB__TIMEOUT=30s
export IS_MOVIE_PROVIDER_TMDB__PRIORITY=1100   # movie sub-provider only
```

### Per-Request Override Headers

Override proxy, API key, or language **for a single request** without restarting. Useful for multi-tenant setups.

| Header | Description |
|--------|-------------|
| `X-Is-Proxy` | Proxy URL for all providers on this request (overrides global env proxy) |
| `X-Is-Api-Key-{PROVIDER}` | Override the API key for a named provider (case-insensitive) |
| `X-Is-Language` | Response language as a BCP 47 tag — e.g. `zh-CN`, `en`, `ja` |

> **Precedence**: per-request header > global `IS_PROVIDER_*` env var.

```sh
# Different TMDB token for this request only
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     "http://localhost:8080/v1/movies/search?q=Inception"

# Route through a proxy for this request
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Proxy: socks5://127.0.0.1:1080" \
     "http://localhost:8080/v1/movies/TMDB/27205"

# Request Chinese metadata
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: zh-CN" \
     "http://localhost:8080/v1/movies/search?q=Inception"
```

### Query Parameters

#### Search endpoints — `/v1/movies/search`, `/v1/actors/search`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `q` | string | **required** | Search keyword. Pass an `http(s)://` URL to trigger a direct detail fetch instead of a keyword search |
| `provider` | string | *(all)* | Restrict to a single provider (case-insensitive). Omit to aggregate all registered providers |
| `fallback` | bool | `true` | Fall back to the local DB cache when the upstream returns no results |

```sh
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/search?q=Inception"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/search?q=Inception&provider=TMDB"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/search?q=https://www.themoviedb.org/movie/27205"
```

#### Detail endpoints — `/v1/movies/:provider/:id`, `/v1/actors/:provider/:id`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `lazy` | bool | `true` | `true` = prefer local cache; `false` = always fetch fresh from upstream and update cache |

```sh
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/TMDB/27205"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/TMDB/27205?lazy=false"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/Trakt/inception"
```

#### Review endpoint — `/v1/reviews/:provider/:id`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `homepage` | string | *(none)* | Scrape reviews from this URL instead of using the stored `id` |
| `lazy` | bool | `true` | Same cache semantics as detail endpoints |

```sh
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/reviews/TMDB/27205"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/reviews/TMDB/27205?lazy=false"
```

---

## Supported Languages

Pass a [BCP 47](https://www.rfc-editor.org/rfc/rfc5646) language tag via the `X-Is-Language` request header. The engine forwards it to each provider using that provider's native format.

| BCP 47 Tag | Language | TMDB | TVDB | AniList | Bangumi | Fanart.tv | TVmaze | Trakt |
|------------|----------|:----:|:----:|:-------:|:-------:|:---------:|:------:|:-----:|
| `zh` / `zh-CN` | Simplified Chinese | ✅ | ✅ | — | ✅ | ✅ | — | — |
| `zh-TW` | Traditional Chinese | ✅ | ✅ | — | ✅ | ✅ | — | — |
| `en` | English | ✅ | ✅ | ✅ | — | ✅ | ✅ | ✅ |
| `ja` | Japanese | ✅ | ✅ | ✅ | — | ✅ | — | — |
| `ko` | Korean | ✅ | ✅ | — | — | — | — | — |
| `fr` | French | ✅ | ✅ | — | — | — | — | — |
| `de` | German | ✅ | ✅ | — | — | — | — | — |
| `es` | Spanish | ✅ | ✅ | — | — | — | — | — |

**Notes:**
- ✅ = provider responds to the language tag; — = provider returns its default language regardless.
- TVmaze and Trakt are English-only APIs.
- Bangumi's content is primarily Chinese and Japanese.
- TMDB accepts any BCP 47 tag supported by its API.
- Both `zh` and `zh-CN` resolve to Simplified Chinese via BCP 47 matching.

---

## API Reference

### Authentication

Private endpoints (marked ✅ in [Endpoint Overview](#endpoint-overview)) require a **Bearer Token**. Public endpoints (`/`, `/healthz`, `/readyz`, `/v1/modules`, `/v1/providers`, `/?redirect=...`) are unaffected.

**Configure the token:**

```sh
./build/inscurascraper-server -token "my-secret-token"
# or
export TOKEN="my-secret-token"
```

When `TOKEN` is empty, authentication is disabled — fine for local development, but set it in production.

> Use a random string: `openssl rand -hex 32`

**Call a private endpoint:**

```sh
curl -H "Authorization: Bearer my-secret-token" \
  "http://localhost:8080/v1/movies/search?q=Inception"
```

Authentication failures return `HTTP 401`:

```json
{ "error": { "code": 401, "message": "unauthorized" } }
```

### Response Format

Every endpoint returns the same envelope:

```json
{ "data": { } }           // success
{ "error": { "code": 400, "message": "..." } }  // failure
```

HTTP status matches `error.code` on failure.

### Endpoint Overview

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/` | — | Service info |
| GET | `/healthz` | — | Liveness probe |
| GET | `/readyz` | — | Readiness probe (checks DB) |
| GET | `/v1/modules` | — | Build dependency list |
| GET | `/v1/providers` | — | Registered providers |
| GET | `/v1/db/version` | ✅ | Database version |
| GET | `/v1/config/proxy` | ✅ | Provider proxy config |
| GET | `/v1/movies/search` | ✅ | Search movies |
| GET | `/v1/movies/:provider/:id` | ✅ | Movie details |
| GET | `/v1/actors/search` | ✅ | Search actors |
| GET | `/v1/actors/:provider/:id` | ✅ | Actor details |
| GET | `/v1/reviews/:provider/:id` | ✅ | Movie reviews |
| GET | `/?redirect=:provider:id` | — | Redirect to upstream page |

### Endpoint Details

<details>
<summary><strong>GET /v1/movies/search</strong></summary>

```sh
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/search?q=Inception&provider=TMDB" | jq
```

```json
{
  "data": [
    {
      "id": "27205", "number": "tt1375666", "title": "Inception",
      "provider": "TMDB", "homepage": "https://www.themoviedb.org/movie/27205",
      "thumb_url": "https://image.tmdb.org/t/p/w300/...jpg",
      "cover_url": "https://image.tmdb.org/t/p/original/...jpg",
      "score": 8.4, "actors": ["Leonardo DiCaprio"], "release_date": "2010-07-15"
    }
  ]
}
```
</details>

<details>
<summary><strong>GET /v1/movies/:provider/:id</strong></summary>

```sh
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/TMDB/27205?lazy=false" | jq
```

```json
{
  "data": {
    "id": "27205", "number": "tt1375666", "title": "Inception",
    "summary": "Cobb, a skilled thief...",
    "provider": "TMDB", "homepage": "https://www.themoviedb.org/movie/27205",
    "director": "Christopher Nolan",
    "actors": ["Leonardo DiCaprio", "Joseph Gordon-Levitt", "Elliot Page"],
    "genres": ["Action", "Science Fiction", "Adventure"],
    "score": 8.4, "runtime": 148, "release_date": "2010-07-15"
  }
}
```
</details>

<details>
<summary><strong>GET /v1/actors/:provider/:id</strong></summary>

```json
{
  "data": {
    "id": "6193", "name": "Leonardo DiCaprio",
    "provider": "TMDB", "homepage": "https://www.themoviedb.org/person/6193",
    "summary": "Leonardo Wilhelm DiCaprio is an American actor...",
    "aliases": ["Leo"], "images": ["https://image.tmdb.org/t/p/original/...jpg"],
    "nationality": "US", "height": 183,
    "birthday": "1974-11-11", "debut_date": "1991-01-01"
  }
}
```
</details>

<details>
<summary><strong>GET /v1/reviews/:provider/:id</strong></summary>

> Only providers implementing the `MovieReviewer` interface support this endpoint; others return 400.

```json
{
  "data": [
    { "title": "A modern classic", "author": "cinemaphile",
      "comment": "Nolan at his peak...", "score": 9.0, "date": "2020-06-01" }
  ]
}
```
</details>

<details>
<summary><strong>GET /v1/providers</strong></summary>

```json
{
  "data": {
    "actor_providers": { "TMDB": "https://www.themoviedb.org", "TVDB": "https://thetvdb.com", "Trakt": "https://trakt.tv" },
    "movie_providers": { "TMDB": "https://www.themoviedb.org", "TVmaze": "https://www.tvmaze.com", "Trakt": "https://trakt.tv" }
  }
}
```
</details>

<details>
<summary><strong>GET /healthz &amp; /readyz</strong></summary>

```sh
curl -s http://localhost:8080/healthz   # {"status":"ok"}
curl -s http://localhost:8080/readyz    # {"status":"ready"} or HTTP 503 if DB unreachable
```
</details>

<details>
<summary><strong>GET /?redirect=TMDB:27205</strong></summary>

Issues a `302` redirect to the upstream homepage for the given provider/ID pair.
</details>

### Error Responses

| HTTP | Meaning |
|------|---------|
| 400 | Bad parameter / malformed ID or URL |
| 401 | Missing or invalid token |
| 404 | Resource or provider not found |
| 500 | Upstream scrape failure / database error |
| 503 | `/readyz`: database unreachable |

---

## Integrations

InscuraScraper exposes a standard HTTP/JSON API. Any tool that can make HTTP requests can use it as a metadata backend.

### tinyMediaManager

tinyMediaManager supports custom URL scrapers. Point your scraper URL to InscuraScraper and pass `Authorization: Bearer <token>` as a custom header. Use `X-Is-Language` to receive metadata in your preferred language.

Example: search endpoint for movies: `http://<host>:8080/v1/movies/search?q={query}`

### Jellyfin / Emby

Use a custom metadata plugin that queries InscuraScraper's REST API. The `/v1/movies/:provider/:id` and `/v1/actors/:provider/:id` endpoints return the metadata fields these platforms require.

### Generic HTTP client

```sh
export SCRAPER=http://localhost:8080
export TOKEN=my-secret-token

# Search with language override
curl -sH "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: zh-CN" \
     "$SCRAPER/v1/movies/search?q=Inception" | jq '.data[0]'

# Fetch details by provider URL
curl -sH "Authorization: Bearer $TOKEN" \
     "$SCRAPER/v1/movies/search?q=https://www.themoviedb.org/movie/27205" | jq
```

---

## Data Models

See [`model/movie.go`](model/movie.go) and [`model/actor.go`](model/actor.go) for full field definitions.

- **`MovieInfo`**: `id, number, title, summary, provider, homepage, director, actors[], thumb_url, big_thumb_url, cover_url, big_cover_url, preview_video_url, preview_video_hls_url, preview_images[], maker, label, series, genres[], score, runtime, release_date`
- **`MovieSearchResult`**: lightweight subset of `MovieInfo`
- **`ActorInfo`**: `id, name, provider, homepage, summary, aliases[], images[], birthday, blood_type, cup_size, measurements, height, nationality, debut_date`
- **`MovieReviewDetail`**: `title, author, comment, score, date`

---

## Development

### Build / Test / Lint

```sh
make              # Dev build → build/inscurascraper-server
make server       # Production build
make lint         # golangci-lint
go test ./...     # Full test suite
```

**Cross-compile:**

```sh
make darwin-arm64 linux-amd64 windows-amd64
make releases     # Emit zips for all architectures under build/
```

### Adding a Provider

See the **Provider Development Guide** in [CLAUDE.md](CLAUDE.md) for the full walkthrough. In brief:

1. Create `provider/<name>/` and embed `*scraper.Scraper`
2. Implement `provider.MovieProvider` and/or `ActorProvider`
3. Call `provider.Register(Name, New)` in `init()`
4. Add a blank import in `engine/register.go`

---

## License & Acknowledgements

Licensed under the [Apache License 2.0](LICENSE).

| Library | Description |
|---------|-------------|
| [gocolly/colly](https://github.com/gocolly/colly) | Elegant scraper and crawler framework for Go |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP web framework |
| [gorm.io/gorm](https://gorm.io/) | ORM for Go |
| [robertkrimen/otto](https://github.com/robertkrimen/otto) | Pure-Go JavaScript interpreter |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | CGo-free SQLite port |
| [antchfx/xpath](https://github.com/antchfx/xpath) | XPath for HTML / XML / JSON |
| [peterbourgon/ff](https://github.com/peterbourgon/ff) | Flags / env unified parsing |
