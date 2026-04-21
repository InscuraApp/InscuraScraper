# InscuraScraper

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md) | **한국어**

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://golang.org/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?logo=github-actions)](.github/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/ghcr.io-inscuraapp%2Finscurascraper-2496ED?logo=docker)](https://github.com/orgs/InscuraApp/packages/container/package/inscurascraper)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

**InscuraScraper** 는 Go 로 작성된 메타데이터 스크래핑 SDK 및 HTTP 서비스입니다. 플러그 가능한 Provider 아키텍처를 통해 TMDB, TVDB, TVmaze, AniList, Fanart.tv 등의 소스에서 영화 및 배우 메타데이터를 수집하고, 통합된 RESTful API 를 제공하며, SQLite 또는 PostgreSQL 을 로컬 캐시로 사용합니다.

> Forked and refactored with the original author's permission.

## 목차

- [특징](#특징)
- [빠른 시작](#빠른-시작)
  - [바이너리](#바이너리)
  - [Docker](#docker)
  - [Docker Compose](#docker-compose)
- [구성](#구성)
  - [서버 옵션](#서버-옵션)
  - [Provider 구성（환경 변수）](#provider-구성환경-변수)
- [API 레퍼런스](#api-레퍼런스)
  - [인증](#인증)
  - [공통 응답 형식](#공통-응답-형식)
  - [선택적 요청 헤더](#선택적-요청-헤더)
  - [엔드포인트 개요](#엔드포인트-개요)
  - [엔드포인트 세부사항](#엔드포인트-세부사항)
- [데이터 모델](#데이터-모델)
- [개발](#개발)
- [기여 / 보안 / 라이선스](#기여--보안--라이선스)

## 특징

- 🔌 **플러그 가능한 Provider 아키텍처**: TMDB, TVDB, TVmaze, AniList, Fanart.tv 내장. 새로운 소스 추가 시 인터페이스 구현 후 등록만 하면 됨
- 🚀 **RESTful API**: Gin 기반, 검색 / 정보 / 리뷰 / 프록시 쿼리용 통합 엔드포인트
- 🗄️ **이중 데이터베이스 지원**: 기본은 인메모리 SQLite(구성 불필요), 프로덕션에서는 PostgreSQL 로 전환 가능
- ⚡ **로컬 캐시**: 먼저 캐시를 조회하고, 없으면 업스트림으로 폴백하여 쿼터 절약
- 🌐 **요청별 커스터마이징**: 요청 헤더로 프록시, API 키, 언어를 동적 전환 가능(재시작 불필요)
- 💊 **관측성**: `/healthz`, `/readyz` 헬스체크 엔드포인트 내장
- 🐳 **크로스플랫폼**: Linux / macOS / Windows / BSD 지원, Dockerfile 및 Docker Compose 포함

## 빠른 시작

> 💡 **바로 사용**: Docker 이미지가 GHCR 에 게시되어 있으므로 `docker pull ghcr.io/inscuraapp/inscurascraper:latest` 만 실행하면 시작할 수 있습니다. [Docker](#docker) 를 참조하세요.

### 바이너리

요구사항: Go 1.25+, `make`.

```sh
git clone https://github.com/InscuraApp/InscuraScraper.git
cd InscuraScraper
make                                  # 결과물: build/inscurascraper-server

./build/inscurascraper-server         # 기본적으로 :8080 리스닝, 인메모리 SQLite 사용
```

검증:

```sh
curl -s http://localhost:8080/healthz
# {"status":"ok"}

curl -s http://localhost:8080/v1/providers | jq
```

### Docker

이미지는 **GitHub Container Registry** 에 게시되어 있습니다: `ghcr.io/inscuraapp/inscurascraper`.

**사용 가능한 태그:**

| 태그 | 의미 |
|------|------|
| `latest` | 최신 안정 버전 |
| `vX.Y.Z` | 특정 버전(프로덕션 환경 권장, 예: `v0.0.1`) |
| `X.Y` | 마이너 버전 라인에 고정(예: `0.0`), 해당 마이너 버전 내 패치 업데이트 자동 수신 |

**지원 아키텍처:** `linux/amd64`, `linux/arm64`

#### 풀 및 실행

```sh
# 최신 버전, 인메모리 SQLite, 인증 없음
docker run --rm -p 8080:8080 \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

#### SQLite 파일 영속화

데이터베이스 파일을 호스트 디렉토리에 마운트하여 컨테이너 재생성 후에도 데이터가 유실되지 않도록 합니다:

```sh
mkdir -p ./data

docker run -d --name inscurascraper -p 8080:8080 \
  -v $PWD/data:/data \
  -e TOKEN=change-me \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest \
  -dsn "/data/inscurascraper.db" -db-auto-migrate
```

#### 로컬 빌드(선택)

사전 빌드된 이미지를 풀하는 대신 소스에서 빌드하고 싶다면:

```sh
docker build -t inscurascraper:local .
docker run --rm -p 8080:8080 inscurascraper:local
```

### Docker Compose

저장소에 `docker-compose.yaml` 이 포함되어 있어, 원커맨드로 InscuraScraper + PostgreSQL 을 기동할 수 있습니다.

> **참고**: 현재 `docker-compose.yaml` 은 기본적으로 로컬 이미지 `inscurascraper-server:latest` 를 사용합니다. GHCR 에 게시된 이미지를 직접 쓰려면 `image:` 를 `ghcr.io/inscuraapp/inscurascraper:latest` 로 변경하면 되며, `docker build` 는 필요하지 않습니다.

```sh
# 옵션 1: GHCR 이미지 사용(권장)
#   docker-compose.yaml 을 편집하여 image: inscurascraper-server:latest 를
#   image: ghcr.io/inscuraapp/inscurascraper:latest 로 변경

# 옵션 2: 로컬 빌드(소스 필요)
docker build -t inscurascraper-server:latest .

# 기동
docker compose up -d

# 로그 확인
docker compose logs -f inscurascraper
```

첫 실행 시 테이블이 자동 생성됩니다(`-db-auto-migrate`). API 토큰은 `docker-compose.yaml` 의 `environment` 섹션에 주입하거나 `.env` 파일을 통해 로드하세요:

```env
IS_PROVIDER_TMDB__API_TOKEN=xxxxx
IS_PROVIDER_FANARTTV__API_KEY=xxxxx
IS_PROVIDER_TVDB__API_KEY=xxxxx
IS_PROVIDER_TVMAZE__API_KEY=xxxxx
```

> **참고**: `docker-compose.yaml` 은 프로젝트 루트의 `./db` 에 PostgreSQL 데이터 볼륨을 마운트합니다. 이 디렉토리는 `.gitignore` 에 포함되어 있으므로 버전 관리에 추가하지 마세요.

## 구성

### 서버 옵션

모든 옵션은 **명령줄 플래그** 또는 **동일 이름의 대문자 환경 변수** 로 설정할 수 있습니다(`peterbourgon/ff` 가 파싱).

| Flag | 환경 변수 | 기본값 | 설명 |
|------|----------|--------|------|
| `-bind` | `BIND` | `""` | 바인드 주소(비어 있으면 모든 인터페이스 리스닝) |
| `-port` | `PORT` | `8080` | HTTP 포트 |
| `-token` | `TOKEN` | `""` | API 인증 토큰. 비어 있으면 인증 비활성화 |
| `-dsn` | `DSN` | `""` | 데이터베이스 DSN. 비어 있으면 인메모리 SQLite |
| `-request-timeout` | `REQUEST_TIMEOUT` | `1m` | 업스트림 요청당 타임아웃 |
| `-db-auto-migrate` | `DB_AUTO_MIGRATE` | `false` | 시작 시 테이블 자동 생성(SQLite 에서는 강제 ON) |
| `-db-prepared-stmt` | `DB_PREPARED_STMT` | `false` | 프리페어드 스테이트먼트 활성화 |
| `-db-max-idle-conns` | `DB_MAX_IDLE_CONNS` | `0` | DB 최대 유휴 커넥션 |
| `-db-max-open-conns` | `DB_MAX_OPEN_CONNS` | `0` | DB 최대 오픈 커넥션 |
| `-version` | `VERSION` | - | 버전 출력 후 종료 |

DSN 예시:

```sh
# SQLite 파일
-dsn "/data/inscurascraper.db"

# PostgreSQL TCP
-dsn "postgres://user:pass@host:5432/inscurascraper?sslmode=disable"

# PostgreSQL Unix socket(docker-compose.yaml 참조)
-dsn "postgres://user:pass@/inscurascraper?host=/var/run/postgresql"
```

### Provider 구성（환경 변수）

각 Provider 의 API 키, 프록시, 우선순위 등은 접두사가 붙은 환경 변수로 주입합니다:

```sh
# actor 와 movie provider 모두에 적용
IS_PROVIDER_{NAME}__{KEY}=value

# actor provider 전용
IS_ACTOR_PROVIDER_{NAME}__{KEY}=value

# movie provider 전용
IS_MOVIE_PROVIDER_{NAME}__{KEY}=value
```

일반적인 `{KEY}`:

| Key | 설명 |
|-----|------|
| `API_TOKEN` / `API_KEY` | 업스트림 API 자격증명 |
| `PRIORITY` | 매칭 우선순위(값이 클수록 우선) |
| `PROXY` | HTTP/SOCKS5 프록시 URL |
| `TIMEOUT` | 요청 타임아웃(Go duration, 예: `30s`) |

예시:

```sh
export IS_PROVIDER_TMDB__API_TOKEN=eyJhbGciOi...
export IS_PROVIDER_TMDB__PRIORITY=10
export IS_PROVIDER_TMDB__PROXY=http://127.0.0.1:7890
```

## API 레퍼런스

### 인증

InscuraScraper 는 **프라이빗 엔드포인트**(아래 [엔드포인트 개요](#엔드포인트-개요) 에서 ✅ 로 표시된 경로)에 간단한 **Bearer Token** 인증 방식을 사용합니다. 퍼블릭 엔드포인트(`/`, `/healthz`, `/readyz`, `/v1/modules`, `/v1/providers`, `/?redirect=...`)는 인증의 영향을 받지 않습니다.

#### 인증 활성화

**명령줄 플래그** 또는 **환경 변수** 중 하나로 Token 을 구성합니다(플래그 우선):

```sh
# 방법 A: 명령줄 플래그
./build/inscurascraper-server -token "my-secret-token"

# 방법 B: 환경 변수
export TOKEN="my-secret-token"
./build/inscurascraper-server
```

**Token 이 비어 있으면(`-token` 이 공백) 인증이 완전히 비활성화**되어 모든 엔드포인트가 공개됩니다. 로컬 개발이나 내부 네트워크 배포에는 적합하지만, 프로덕션 환경에서는 반드시 명시적으로 설정하세요.

Docker 상황:

```sh
docker run -d -p 8080:8080 \
  -e TOKEN=my-secret-token \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

Docker Compose 상황 —— `docker-compose.yaml` 의 `environment` 섹션에 추가:

```yaml
services:
  inscurascraper:
    environment:
      TOKEN: my-secret-token
```

또는 저장소 루트의 `.env` 파일을 통해 로드:

```env
TOKEN=my-secret-token
```

> 💡 충분한 길이의 난수 문자열(예: `openssl rand -hex 32`)을 시크릿 관리 도구를 통해 주입할 것을 권장합니다. 저장소나 이미지에 평문으로 기록하지 마세요.

#### 프라이빗 엔드포인트 호출

요청 헤더에 Token 을 첨부하며, **형식은 반드시 `Bearer <token>`**(대소문자 구분):

```sh
curl -H "Authorization: Bearer my-secret-token" \
  "http://localhost:8080/v1/movies/search?q=Inception"
```

검증 실패 시 항상 반환:

```
HTTP/1.1 401 Unauthorized
```

```json
{ "error": { "code": 401, "message": "unauthorized" } }
```

흔한 원인:

- `Authorization` 헤더 누락
- 접두사가 `Bearer` 가 아님(대소문자 구분, `bearer`, `Token` 등은 거부)
- Token 값이 서버 측 구성과 불일치

#### Token 교체 또는 폐기

현재 구현은 **단일 Token 정적 구성** 입니다. Token 변경 시 새 값을 적용하려면 프로세스 재시작이 필요합니다. 다중 Token 관리 또는 동적 폐기가 필요하다면 `route/auth.TokenStore` 를 기반으로 코드에서 확장할 수 있습니다.

### 공통 응답 형식

모든 엔드포인트는 다음 형식으로 반환합니다:

```json
{
  "data": { },
  "error": { "code": 400, "message": "..." }
}
```

- 성공: `data` 만 반환
- 실패: `error` 만 반환하며, HTTP 상태 코드는 `error.code` 와 일치

### 선택적 요청 헤더

각 요청마다 Provider 동작을 재정의할 수 있으며, 재시작이 필요하지 않습니다:

| Header | 설명 |
|--------|------|
| `X-Is-Proxy` | 이번 요청에 대해 모든 Provider 가 사용할 프록시 URL |
| `X-Is-Api-Key-{PROVIDER}` | 지정된 Provider 의 API 키 재정의(대소문자 무시) |
| `X-Is-Language` | 응답 언어, BCP 47 태그(예: `ko-KR`, `en-US`) |

예시:

```sh
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: ko-KR" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     "http://localhost:8080/v1/movies/search?q=Inception"
```

### 엔드포인트 개요

| Method | Path | 인증 | 설명 |
|--------|------|------|------|
| GET | `/` | ❌ | 서비스 정보 |
| GET | `/healthz` | ❌ | Liveness 프로브 |
| GET | `/readyz` | ❌ | Readiness 프로브(DB 검사) |
| GET | `/v1/modules` | ❌ | 빌드 의존성 목록 |
| GET | `/v1/providers` | ❌ | 등록된 Provider 목록 |
| GET | `/v1/db/version` | ✅ | 데이터베이스 버전 |
| GET | `/v1/config/proxy` | ✅ | 현재 Provider 프록시 구성 |
| GET | `/v1/movies/search` | ✅ | 영화 검색 |
| GET | `/v1/movies/:provider/:id` | ✅ | 영화 상세 조회 |
| GET | `/v1/actors/search` | ✅ | 배우 검색 |
| GET | `/v1/actors/:provider/:id` | ✅ | 배우 상세 조회 |
| GET | `/v1/reviews/:provider/:id` | ✅ | 영화 리뷰 조회 |
| GET | `/?redirect=:provider:id` | ❌ | providerID 기반 소스 사이트 홈 리다이렉트 |

### 엔드포인트 세부사항

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
# DB 도달 불가 시: HTTP 503 {"status":"not_ready","error":"..."}
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

쿼리 파라미터:

| 파라미터 | 필수 | 설명 |
|---------|------|------|
| `q` | ✅ | 키워드. http(s) URL 을 전달하면 Provider 와 ID 를 자동 파싱하여 상세를 가져옴 |
| `provider` | ❌ | Provider 제한(대소문자 무시). 미지정 시 모든 Provider 집계 |
| `fallback` | ❌ | 업스트림 결과가 없을 때 로컬 DB 캐시로 폴백 여부. 기본값 `true` |

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

쿼리 파라미터:

| 파라미터 | 설명 |
|---------|------|
| `lazy` | `true`(기본) = 캐시 우선; `false` = 업스트림에서 강제 재조회 |

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

파라미터는 영화 엔드포인트와 동일. 배우 응답 예시:

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

쿼리 파라미터:

| 파라미터 | 설명 |
|---------|------|
| `homepage` | 선택. URL 로 직접 리뷰 스크래핑 |
| `lazy` | 위와 동일 |

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

> `MovieReviewer` 인터페이스를 구현한 Provider 만 이 엔드포인트를 지원합니다. 그 외는 400 을 반환합니다.

#### `GET /v1/db/version`

```json
{ "data": { "version": "PostgreSQL 15.6 on x86_64-pc-linux-musl ..." } }
```

#### `GET /v1/config/proxy`

각 Provider 의 현재 영속화된 프록시 설정을 반환합니다(환경 변수로 주입되며 런타임에는 읽기 전용).

```json
{
  "data": {
    "TMDB":   "http://127.0.0.1:7890",
    "TVDB":   ""
  }
}
```

#### `GET /?redirect=TMDB:27205`

해당 영화 / 배우의 소스 사이트 홈으로 `302` 리다이렉트합니다.

### 에러 응답

```json
{
  "error": {
    "code": 404,
    "message": "info not found"
  }
}
```

일반적인 상태 코드:

| HTTP | 의미 |
|------|------|
| 400 | 파라미터 오류 / ID 또는 URL 형식 오류 |
| 401 | 토큰 누락 또는 불법 |
| 404 | 해당 리소스 또는 Provider 없음 |
| 500 | 업스트림 스크래핑 실패 / 데이터베이스 오류 |
| 503 | `/readyz` 데이터베이스 도달 불가 |

## 데이터 모델

전체 필드 정의는 [`model/movie.go`](model/movie.go) 와 [`model/actor.go`](model/actor.go) 를 참조. 요약:

- `MovieInfo`: `id, number, title, summary, provider, homepage, director, actors[], thumb_url, big_thumb_url, cover_url, big_cover_url, preview_video_url, preview_video_hls_url, preview_images[], maker, label, series, genres[], score, runtime, release_date`
- `MovieSearchResult`: `MovieInfo` 의 경량 서브셋
- `ActorInfo`: `id, name, provider, homepage, summary, aliases[], images[], birthday, blood_type, cup_size, measurements, height, nationality, debut_date`
- `MovieReviewDetail`: `title, author, comment, score, date`

## 개발

### 빌드 / 테스트 / Lint

```sh
make              # 개발 빌드
make server       # 프로덕션 빌드
make lint         # golangci-lint
go test ./...     # 전체 단위 테스트
```

크로스 컴파일:

```sh
make darwin-arm64 linux-amd64 windows-amd64
make releases          # 모든 아키텍처의 zip 을 build/ 에 출력
```

### 새로운 Provider 개발

자세한 가이드는 [CLAUDE.md](CLAUDE.md) 의 **Provider Development Guide** 와 [CONTRIBUTING.md](CONTRIBUTING.md) 를 참조하세요. 간단한 절차:

1. `provider/<name>/` 아래에 디렉토리를 만들고 `*scraper.Scraper` 를 임베드
2. `provider.MovieProvider` 및/또는 `ActorProvider` 구현
3. `init()` 에서 `provider.Register(Name, New)` 호출
4. `engine/register.go` 에 blank import 추가

## 기여 / 보안 / 라이선스

- [기여 가이드](CONTRIBUTING.md)
- [행동 강령](CODE_OF_CONDUCT.md)
- [보안 공개](SECURITY.md)(공개 Issue 를 제기하지 마세요)
- [변경 로그](CHANGELOG.md)
- 라이선스: [Apache 2.0](LICENSE)

## 감사의 글

| Library | Description |
|---------|-------------|
| [gocolly/colly](https://github.com/gocolly/colly) | Elegant scraper and crawler framework for Go |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP web framework |
| [gorm.io/gorm](https://gorm.io/) | ORM for Go |
| [robertkrimen/otto](https://github.com/robertkrimen/otto) | Pure-Go JavaScript interpreter |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | CGo-free SQLite port |
| [antchfx/xpath](https://github.com/antchfx/xpath) | XPath for HTML / XML / JSON |
| [peterbourgon/ff](https://github.com/peterbourgon/ff) | Flags / env unified parsing |
