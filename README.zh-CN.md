# InscuraScraper

[English](README.md) | **简体中文**

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://golang.org/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?logo=github-actions)](.github/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/ghcr.io-inscuraapp%2Finscurascraper-2496ED?logo=docker)](https://github.com/orgs/InscuraApp/packages/container/package/inscurascraper)

**InscuraScraper** 是一个用 Go 编写的元数据抓取 SDK 与 HTTP 服务。它通过可插拔的 Provider 机制从 TMDB、TVDB、TVmaze、AniList、Fanart.tv、Trakt 等源抓取影片与演员元数据，提供统一的 RESTful API，并使用 SQLite 或 PostgreSQL 作为本地缓存。

> Forked and refactored with the original author's permission.

## 目录

- [运行方式](#运行方式)
  - [Docker](#docker)
  - [Docker Compose](#docker-compose)
  - [二进制方式](#二进制方式)
- [配置](#配置)
  - [服务器参数](#服务器参数)
  - [Provider 环境变量](#provider-环境变量)
  - [每请求覆盖请求头](#每请求覆盖请求头)
  - [Query 参数](#query-参数)
- [支持的语言](#支持的语言)
- [API 参考](#api-参考)
  - [鉴权](#鉴权)
  - [响应格式](#响应格式)
  - [端点一览](#端点一览)
  - [端点详情](#端点详情)
  - [错误响应](#错误响应)
- [整合](#整合)
- [数据模型](#数据模型)
- [开发](#开发)
- [许可证与致谢](#许可证与致谢)

---

## 运行方式

### Docker

镜像已发布到 **GitHub Container Registry**：`ghcr.io/inscuraapp/inscurascraper`

**可用 Tag：**

| Tag | 含义 |
|-----|------|
| `latest` | 最新稳定版本 |
| `vX.Y.Z` | 指定版本（推荐生产环境，如 `v0.0.1`） |
| `X.Y` | 锁定次版本线（如 `0.0`），自动接收补丁更新 |

**支持架构：** `linux/amd64`、`linux/arm64`

**最快启动 — 内存 SQLite，无鉴权：**

```sh
docker run --rm -p 8080:8080 \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

**持久化 SQLite（容器重建后数据不丢失）：**

```sh
mkdir -p ./data
docker run -d --name inscurascraper -p 8080:8080 \
  -v $PWD/data:/data \
  -e TOKEN=change-me \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest \
  -dsn "/data/inscurascraper.db" -db-auto-migrate
```

**本地构建镜像（可选）：**

```sh
docker build -t inscurascraper:local .
docker run --rm -p 8080:8080 inscurascraper:local
```

**验证：**

```sh
curl -s http://localhost:8080/healthz   # {"status":"ok"}
curl -s http://localhost:8080/v1/providers | jq
```

### Docker Compose

仓库已提供 `docker-compose.yaml`，一键启动 InscuraScraper + PostgreSQL。

> **注意**：`docker-compose.yaml` 默认使用本地镜像 `inscurascraper-server:latest`。如需直接使用 GHCR 镜像，将 `image:` 改为 `ghcr.io/inscuraapp/inscurascraper:latest` 即可，无需先行构建。

```sh
# 方式 A：直接使用 GHCR 镜像（推荐，无需构建）
#   编辑 docker-compose.yaml，将 image 改为 ghcr.io/inscuraapp/inscurascraper:latest

# 方式 B：本地构建
docker build -t inscurascraper-server:latest .

docker compose up -d
docker compose logs -f inscurascraper
```

通过 `.env` 文件或 `docker-compose.yaml` 的 `environment` 段注入 API Token：

```env
TOKEN=change-me
IS_PROVIDER_TMDB__API_TOKEN=xxxxx
IS_PROVIDER_FANARTTV__API_KEY=xxxxx
IS_PROVIDER_TVDB__API_KEY=xxxxx
IS_PROVIDER_TRAKT__CLIENT_ID=xxxxx
```

> **注意**：PostgreSQL 数据挂载到 `./db`，已加入 `.gitignore`，请勿提交至版本控制。

### 二进制方式

前置条件：Go 1.25+、`make`。

```sh
git clone https://github.com/InscuraApp/InscuraScraper.git
cd InscuraScraper
make                              # 产物：build/inscurascraper-server
./build/inscurascraper-server     # 监听 :8080，使用内存 SQLite
```

---

## 配置

### 服务器参数

所有参数均可通过 **命令行 Flag** 或 **同名大写环境变量** 设置（由 `peterbourgon/ff` 解析），Flag 优先级更高。

| Flag | 环境变量 | 默认值 | 说明 |
|------|---------|--------|------|
| `-bind` | `BIND` | `""` | 绑定地址（留空监听所有网卡） |
| `-port` | `PORT` | `8080` | HTTP 端口 |
| `-token` | `TOKEN` | `""` | API 鉴权 Token；留空则关闭鉴权 |
| `-dsn` | `DSN` | `""` | 数据库 DSN；留空则使用内存 SQLite |
| `-request-timeout` | `REQUEST_TIMEOUT` | `1m` | 单次上游请求超时 |
| `-db-auto-migrate` | `DB_AUTO_MIGRATE` | `false` | 启动时自动建表（SQLite 强制开启） |
| `-db-prepared-stmt` | `DB_PREPARED_STMT` | `false` | 启用预编译语句 |
| `-db-max-idle-conns` | `DB_MAX_IDLE_CONNS` | `0` | 最大空闲连接数 |
| `-db-max-open-conns` | `DB_MAX_OPEN_CONNS` | `0` | 最大打开连接数 |
| `-version` | — | — | 打印版本后退出 |

**DSN 示例：**

```sh
-dsn "/data/inscurascraper.db"                                         # SQLite 文件
-dsn "postgres://user:pass@host:5432/inscurascraper?sslmode=disable"  # PostgreSQL TCP
-dsn "postgres://user:pass@/inscurascraper?host=/var/run/postgresql"   # PostgreSQL Unix socket
```

### Provider 环境变量

每个 Provider 的 API Key、代理、优先级在启动时通过带前缀的环境变量注入，属于**全局配置**。单请求级别的覆盖请使用[每请求覆盖请求头](#每请求覆盖请求头)。

**命名规则：**

```
IS_PROVIDER_{NAME}__{KEY}=value           # 同时作用于 actor 和 movie 子 Provider
IS_ACTOR_PROVIDER_{NAME}__{KEY}=value     # 仅作用于 actor 子 Provider
IS_MOVIE_PROVIDER_{NAME}__{KEY}=value     # 仅作用于 movie 子 Provider
```

`{NAME}` 为 Provider 名称的**大写形式**（如 `TMDB`、`TVDB`、`TRAKT`）。

**通用键（适用于任意 Provider）：**

| Key | 类型 | 说明 |
|-----|------|------|
| `PRIORITY` | float | 匹配优先级，数值越大越优先 |
| `PROXY` | string | HTTP 或 SOCKS5 代理，如 `http://127.0.0.1:7890` 或 `socks5://127.0.0.1:1080` |
| `TIMEOUT` | duration | 单次上游请求超时，Go duration 格式，如 `30s`、`2m` |

**内置 Provider 凭证键：**

| Provider | 环境变量 | 是否必填 | 获取地址 |
|----------|---------|---------|---------|
| **TMDB** | `IS_PROVIDER_TMDB__API_TOKEN` | 是 | [themoviedb.org/settings/api](https://www.themoviedb.org/settings/api) — 使用 **Bearer Token（v4 鉴权）** |
| **TVDB** | `IS_PROVIDER_TVDB__API_KEY` | 是 | [thetvdb.com/api-information](https://thetvdb.com/api-information) |
| **Fanart.tv** | `IS_PROVIDER_FANARTTV__API_KEY` | 是 | [fanart.tv/get-an-api-key](https://fanart.tv/get-an-api-key/) |
| **Trakt** | `IS_PROVIDER_TRAKT__CLIENT_ID` | 是 | [trakt.tv/oauth/applications](https://trakt.tv/oauth/applications) — 注册应用后复制 **Client ID** |
| **AniList** | *(无)* | — | 公开 API，无需 Key |
| **TVmaze** | *(无)* | — | 公开 API，无需 Key |
| **Bangumi** | *(无)* | — | 公开 API，无需 Key |

**完整示例：**

```sh
export IS_PROVIDER_TMDB__API_TOKEN=eyJhbGciOiJSUzI1NiJ9...
export IS_PROVIDER_TVDB__API_KEY=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
export IS_PROVIDER_FANARTTV__API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export IS_PROVIDER_TRAKT__CLIENT_ID=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

export IS_PROVIDER_TMDB__PRIORITY=1000
export IS_PROVIDER_TMDB__PROXY=http://127.0.0.1:7890
export IS_PROVIDER_TMDB__TIMEOUT=30s
export IS_MOVIE_PROVIDER_TMDB__PRIORITY=1100   # 仅针对 TMDB 的 movie 子 Provider
```

### 每请求覆盖请求头

通过以下请求头，可在**单次请求**中覆盖代理、API Key 或响应语言，无需重启服务，适用于多租户场景。

| Header | 说明 |
|--------|------|
| `X-Is-Proxy` | 本次请求所有 Provider 使用的代理 URL（覆盖全局代理） |
| `X-Is-Api-Key-{PROVIDER}` | 覆盖指定 Provider 的 API Key（Provider 名大小写不敏感） |
| `X-Is-Language` | 响应语言，BCP 47 标签，如 `zh-CN`、`en`、`ja` |

> **优先级**：请求头 > 全局 `IS_PROVIDER_*` 环境变量。

```sh
# 本次请求使用不同的 TMDB Token
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     "http://localhost:8080/v1/movies/search?q=Inception"

# 本次请求通过指定代理访问所有上游
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Proxy: socks5://127.0.0.1:1080" \
     "http://localhost:8080/v1/movies/TMDB/27205"

# 请求中文元数据
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: zh-CN" \
     "http://localhost:8080/v1/movies/search?q=Inception"
```

### Query 参数

#### 搜索端点 — `/v1/movies/search`、`/v1/actors/search`

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `q` | string | **必填** | 搜索关键词。传入 `http(s)://` 格式的 URL 时，自动解析 Provider 和 ID，直接返回详情而非搜索结果 |
| `provider` | string | *(全部)* | 限定到单个 Provider（大小写不敏感）。不传则聚合并去重所有已注册 Provider 的结果 |
| `fallback` | bool | `true` | 上游无结果时是否回退到本地 DB 缓存 |

```sh
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/search?q=Inception"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/search?q=Inception&provider=TMDB"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/search?q=https://www.themoviedb.org/movie/27205"
```

#### 详情端点 — `/v1/movies/:provider/:id`、`/v1/actors/:provider/:id`

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `lazy` | bool | `true` | `true` = 优先读本地缓存，无缓存才回源；`false` = 强制回源拉取最新数据并更新缓存 |

```sh
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/TMDB/27205"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/TMDB/27205?lazy=false"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/movies/Trakt/inception"
```

#### 评论端点 — `/v1/reviews/:provider/:id`

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `homepage` | string | *(无)* | 直接按此 URL 抓取评论，而不使用路径中的 `id` |
| `lazy` | bool | `true` | 与详情端点相同的缓存语义 |

```sh
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/reviews/TMDB/27205"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/reviews/TMDB/27205?lazy=false"
```

---

## 支持的语言

通过 `X-Is-Language` 请求头传入 [BCP 47](https://www.rfc-editor.org/rfc/rfc5646) 语言标签。引擎会将其转换为各 Provider 的原生格式后传递。

| BCP 47 标签 | 语言 | TMDB | TVDB | AniList | Bangumi | Fanart.tv | TVmaze | Trakt |
|------------|------|:----:|:----:|:-------:|:-------:|:---------:|:------:|:-----:|
| `zh` / `zh-CN` | 简体中文 | ✅ | ✅ | — | ✅ | ✅ | — | — |
| `zh-TW` | 繁体中文 | ✅ | ✅ | — | ✅ | ✅ | — | — |
| `en` | 英文 | ✅ | ✅ | ✅ | — | ✅ | ✅ | ✅ |
| `ja` | 日文 | ✅ | ✅ | ✅ | — | ✅ | — | — |
| `ko` | 韩文 | ✅ | ✅ | — | — | — | — | — |
| `fr` | 法文 | ✅ | ✅ | — | — | — | — | — |
| `de` | 德文 | ✅ | ✅ | — | — | — | — | — |
| `es` | 西班牙文 | ✅ | ✅ | — | — | — | — | — |

**说明：**
- ✅ = Provider 会根据该标签返回对应语言的内容；— = Provider 忽略语言设置，始终返回其默认语言。
- TVmaze 和 Trakt 为纯英文 API。
- Bangumi 内容主要为中文和日文。
- TMDB 支持其 API 接受的任意 BCP 47 标签。
- `zh` 和 `zh-CN` 通过 BCP 47 匹配算法均解析为简体中文。

---

## API 参考

### 鉴权

私有端点（参考下文[端点一览](#端点一览)中 ✅ 标记的路径）采用 **Bearer Token** 鉴权。公开端点（`/`、`/healthz`、`/readyz`、`/v1/modules`、`/v1/providers`、`/?redirect=...`）不受影响。

**配置 Token：**

```sh
./build/inscurascraper-server -token "my-secret-token"
# 或
export TOKEN="my-secret-token"
```

**`TOKEN` 为空时鉴权整体关闭**，适合本地开发；生产环境务必显式配置。

> 建议使用随机串：`openssl rand -hex 32`

**调用私有端点：**

```sh
curl -H "Authorization: Bearer my-secret-token" \
  "http://localhost:8080/v1/movies/search?q=Inception"
```

鉴权失败统一返回 `HTTP 401`：

```json
{ "error": { "code": 401, "message": "unauthorized" } }
```

### 响应格式

所有端点统一使用同一信封格式：

```json
{ "data": { } }                                   // 成功
{ "error": { "code": 400, "message": "..." } }    // 失败
```

失败时 HTTP 状态码与 `error.code` 对应。

### 端点一览

| Method | Path | 鉴权 | 说明 |
|--------|------|:----:|------|
| GET | `/` | — | 服务信息 |
| GET | `/healthz` | — | 存活探针 |
| GET | `/readyz` | — | 就绪探针（检测数据库） |
| GET | `/v1/modules` | — | 构建依赖列表 |
| GET | `/v1/providers` | — | 已注册 Provider 列表 |
| GET | `/v1/db/version` | ✅ | 数据库版本 |
| GET | `/v1/config/proxy` | ✅ | Provider 代理配置 |
| GET | `/v1/movies/search` | ✅ | 搜索影片 |
| GET | `/v1/movies/:provider/:id` | ✅ | 获取影片详情 |
| GET | `/v1/actors/search` | ✅ | 搜索演员 |
| GET | `/v1/actors/:provider/:id` | ✅ | 获取演员详情 |
| GET | `/v1/reviews/:provider/:id` | ✅ | 获取影片评论 |
| GET | `/?redirect=:provider:id` | — | 重定向到源站主页 |

### 端点详情

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

> 只有实现了 `MovieReviewer` 接口的 Provider 才支持此端点；其他 Provider 返回 400。

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
curl -s http://localhost:8080/readyz    # {"status":"ready"} 或数据库不可用时返回 HTTP 503
```
</details>

<details>
<summary><strong>GET /?redirect=TMDB:27205</strong></summary>

直接 `302` 重定向到该影片 / 演员在源站的主页。
</details>

### 错误响应

| HTTP | 含义 |
|------|------|
| 400 | 参数错误 / ID 或 URL 格式错误 |
| 401 | 缺少或非法 Token |
| 404 | 未找到对应资源或 Provider |
| 500 | 上游抓取失败 / 数据库错误 |
| 503 | `/readyz` 数据库不可用 |

---

## 整合

InscuraScraper 提供标准的 HTTP/JSON API，任何支持 HTTP 请求的工具均可将其作为元数据后端。

### tinyMediaManager

tinyMediaManager 支持自定义 URL 刮削器。将刮削器地址指向 InscuraScraper，并在自定义请求头中配置 `Authorization: Bearer <token>`。通过 `X-Is-Language` 可按需获取指定语言的元数据。

搜索接口示例：`http://<host>:8080/v1/movies/search?q={query}`

### Jellyfin / Emby

可通过自定义元数据插件调用 InscuraScraper 的 REST API。`/v1/movies/:provider/:id` 和 `/v1/actors/:provider/:id` 返回这些平台所需的标准元数据字段。

### 通用 HTTP 客户端

```sh
export SCRAPER=http://localhost:8080
export TOKEN=my-secret-token

# 带语言的搜索
curl -sH "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: zh-CN" \
     "$SCRAPER/v1/movies/search?q=Inception" | jq '.data[0]'

# 通过 Provider 页面 URL 直接获取详情
curl -sH "Authorization: Bearer $TOKEN" \
     "$SCRAPER/v1/movies/search?q=https://www.themoviedb.org/movie/27205" | jq
```

---

## 数据模型

完整字段定义见 [`model/movie.go`](model/movie.go) 与 [`model/actor.go`](model/actor.go)。

- **`MovieInfo`**：`id, number, title, summary, provider, homepage, director, actors[], thumb_url, big_thumb_url, cover_url, big_cover_url, preview_video_url, preview_video_hls_url, preview_images[], maker, label, series, genres[], score, runtime, release_date`
- **`MovieSearchResult`**：`MovieInfo` 的轻量子集
- **`ActorInfo`**：`id, name, provider, homepage, summary, aliases[], images[], birthday, blood_type, cup_size, measurements, height, nationality, debut_date`
- **`MovieReviewDetail`**：`title, author, comment, score, date`

---

## 开发

### 构建 / 测试 / Lint

```sh
make              # 开发构建 → build/inscurascraper-server
make server       # 生产构建
make lint         # golangci-lint
go test ./...     # 全量单测
```

**交叉编译：**

```sh
make darwin-arm64 linux-amd64 windows-amd64
make releases     # 输出所有架构的 zip 到 build/
```

### 开发新 Provider

详细指南见 [CLAUDE.md](CLAUDE.md) 的 **Provider Development Guide**。简要步骤：

1. 在 `provider/<name>/` 下创建目录，嵌入 `*scraper.Scraper`
2. 实现 `provider.MovieProvider` 和/或 `ActorProvider`
3. 在 `init()` 中调用 `provider.Register(Name, New)`
4. 在 `engine/register.go` 中添加 blank import

---

## 许可证与致谢

本项目采用 [Apache License 2.0](LICENSE) 许可。

| Library | Description |
|---------|-------------|
| [gocolly/colly](https://github.com/gocolly/colly) | Elegant scraper and crawler framework for Go |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP web framework |
| [gorm.io/gorm](https://gorm.io/) | ORM for Go |
| [robertkrimen/otto](https://github.com/robertkrimen/otto) | Pure-Go JavaScript interpreter |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | CGo-free SQLite port |
| [antchfx/xpath](https://github.com/antchfx/xpath) | XPath for HTML / XML / JSON |
| [peterbourgon/ff](https://github.com/peterbourgon/ff) | Flags / env unified parsing |
