# InscuraScraper

[English](README.md) | **简体中文**

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://golang.org/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?logo=github-actions)](.github/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/ghcr.io-inscuraapp%2Finscurascraper-2496ED?logo=docker)](https://github.com/orgs/InscuraApp/packages/container/package/inscurascraper)

**InscuraScraper** 是一个用 Go 编写的元数据抓取 SDK 与 HTTP 服务。它通过可插拔的 Provider 机制从 TMDB、TVDB、TVmaze、AniList、Fanart.tv 等源抓取影片与演员元数据，提供统一的 RESTful API，并使用 SQLite 或 PostgreSQL 作为本地缓存。

> Forked and refactored with the original author's permission.

## 目录

- [特性](#特性)
- [快速开始](#快速开始)
  - [二进制方式](#二进制方式)
  - [Docker 方式](#docker-方式)
  - [Docker Compose 方式](#docker-compose-方式)
- [配置](#配置)
  - [服务器参数](#服务器参数)
  - [Provider 配置（环境变量）](#provider-配置环境变量)
- [API 参考](#api-参考)
  - [鉴权](#鉴权)
  - [通用响应格式](#通用响应格式)
  - [可选请求头](#可选请求头)
  - [端点一览](#端点一览)
  - [端点详情](#端点详情)
- [数据模型](#数据模型)
- [开发](#开发)
- [贡献 / 安全 / 许可证](#贡献--安全--许可证)

## 特性

- 🔌 **可插拔 Provider 架构**：已内置 TMDB、TVDB、TVmaze、AniList、Fanart.tv，开发新源只需实现接口并注册
- 🚀 **RESTful API**：Gin 驱动，统一的搜索 / 信息 / 评论 / 代理查询端点
- 🗄️ **双数据库支持**：默认内存 SQLite（零配置），生产可切到 PostgreSQL
- ⚡ **本地缓存**：先查缓存再回源，降低上游配额消耗
- 🌐 **每请求定制**：通过请求头动态切换代理、API Key、语言,无需重启
- 💊 **可观测性**：内置 `/healthz`、`/readyz` 健康检查端点
- 🐳 **多平台**：Linux / macOS / Windows / BSD，已提供 Dockerfile 与 Docker Compose

## 快速开始

> 💡 **开箱即用**：Docker 镜像已发布到 GHCR，执行 `docker pull ghcr.io/inscuraapp/inscurascraper:latest` 即可开始使用，详见 [Docker 方式](#docker-方式)。

### 二进制方式

前置：Go 1.25+、`make`。

```sh
git clone https://github.com/InscuraApp/InscuraScraper.git
cd InscuraScraper
make                                  # 产物：build/inscurascraper-server

./build/inscurascraper-server         # 默认监听 :8080，使用内存 SQLite
```

验证：

```sh
curl -s http://localhost:8080/healthz
# {"status":"ok"}

curl -s http://localhost:8080/v1/providers | jq
```

### Docker 方式

镜像已发布到 **GitHub Container Registry**：`ghcr.io/inscuraapp/inscurascraper`。

**可用 tag：**

| Tag | 含义 |
|-----|------|
| `latest` | 最新的稳定版本 |
| `vX.Y.Z` | 指定版本（推荐生产环境使用，例如 `v0.0.1`） |
| `X.Y` | 锁定到次版本线（例如 `0.0`），自动获取该次版本内的补丁更新 |

**支持架构：** `linux/amd64`、`linux/arm64`

#### 拉取并运行

```sh
# 最新版本，内存 SQLite，无鉴权
docker run --rm -p 8080:8080 \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

#### 带持久化 SQLite 文件

将数据库文件挂载到宿主机目录，避免容器重建后数据丢失：

```sh
mkdir -p ./data

docker run -d --name inscurascraper -p 8080:8080 \
  -v $PWD/data:/data \
  -e TOKEN=change-me \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest \
  -dsn "/data/inscurascraper.db" -db-auto-migrate
```

#### 本地自行构建（可选）

如果你想从源码构建而不是拉取预构建镜像：

```sh
docker build -t inscurascraper:local .
docker run --rm -p 8080:8080 inscurascraper:local
```

### Docker Compose 方式

仓库已提供 `docker-compose.yaml`，一键启动 InscuraScraper + PostgreSQL。

> **注意**：当前 `docker-compose.yaml` 默认使用本地镜像 `inscurascraper-server:latest`。如果你希望直接使用 GHCR 发布的镜像，将 `image:` 改为 `ghcr.io/inscuraapp/inscurascraper:latest` 即可，无需先行 `docker build`。

```sh
# 选项 1：直接使用 GHCR 镜像（推荐）
#   编辑 docker-compose.yaml，将 image: inscurascraper-server:latest
#   改为 image: ghcr.io/inscuraapp/inscurascraper:latest

# 选项 2：本地构建镜像（需要源码）
docker build -t inscurascraper-server:latest .

# 启动
docker compose up -d

# 查看日志
docker compose logs -f inscurascraper
```

首次启动会自动建表（`-db-auto-migrate`）。将你的 API Token 注入到 `docker-compose.yaml` 的 `environment` 段落，或通过 `.env` 文件加载：

```env
IS_PROVIDER_TMDB__API_TOKEN=xxxxx
IS_PROVIDER_FANARTTV__API_KEY=xxxxx
IS_PROVIDER_TVDB__API_KEY=xxxxx
IS_PROVIDER_TVMAZE__API_KEY=xxxxx
```

> **注意**：`docker-compose.yaml` 将 PostgreSQL 数据卷挂载到项目目录的 `./db`，该目录已通过 `.gitignore` 排除，请勿将其加入版本控制。

## 配置

### 服务器参数

所有参数可通过 **命令行 flag** 或 **同名大写环境变量** 设置（由 `peterbourgon/ff` 解析）。

| Flag | 环境变量 | 默认值 | 说明 |
|------|---------|--------|------|
| `-bind` | `BIND` | `""` | 绑定地址（留空监听所有网卡） |
| `-port` | `PORT` | `8080` | HTTP 端口 |
| `-token` | `TOKEN` | `""` | API 鉴权 Token；留空则关闭鉴权 |
| `-dsn` | `DSN` | `""` | 数据库 DSN；留空则使用内存 SQLite |
| `-request-timeout` | `REQUEST_TIMEOUT` | `1m` | 单次上游请求超时 |
| `-db-auto-migrate` | `DB_AUTO_MIGRATE` | `false` | 启动时自动建表（SQLite 强制开启） |
| `-db-prepared-stmt` | `DB_PREPARED_STMT` | `false` | 启用预编译语句 |
| `-db-max-idle-conns` | `DB_MAX_IDLE_CONNS` | `0` | 最大空闲连接 |
| `-db-max-open-conns` | `DB_MAX_OPEN_CONNS` | `0` | 最大打开连接 |
| `-version` | `VERSION` | - | 打印版本后退出 |

DSN 示例：

```sh
# SQLite 文件
-dsn "/data/inscurascraper.db"

# PostgreSQL TCP
-dsn "postgres://user:pass@host:5432/inscurascraper?sslmode=disable"

# PostgreSQL Unix socket（见 docker-compose.yaml）
-dsn "postgres://user:pass@/inscurascraper?host=/var/run/postgresql"
```

### Provider 配置（环境变量）

每个 Provider 的 API Key、代理、优先级等通过带前缀的环境变量注入：

```sh
# 同时作用于 actor 和 movie provider
IS_PROVIDER_{NAME}__{KEY}=value

# 仅作用于 actor provider
IS_ACTOR_PROVIDER_{NAME}__{KEY}=value

# 仅作用于 movie provider
IS_MOVIE_PROVIDER_{NAME}__{KEY}=value
```

常用 `{KEY}`：

| Key | 说明 |
|-----|------|
| `API_TOKEN` / `API_KEY` | 上游 API 凭证 |
| `PRIORITY` | 匹配优先级（数值越大越优先） |
| `PROXY` | HTTP/SOCKS5 代理 URL |
| `TIMEOUT` | 请求超时（Go duration，如 `30s`） |

示例：

```sh
export IS_PROVIDER_TMDB__API_TOKEN=eyJhbGciOi...
export IS_PROVIDER_TMDB__PRIORITY=10
export IS_PROVIDER_TMDB__PROXY=http://127.0.0.1:7890
```

## API 参考

### 鉴权

InscuraScraper 对 **私有端点**（参考下文 [端点一览](#端点一览) 中 ✅ 标记的路径）采用简单的 **Bearer Token** 鉴权。公开端点（`/`、`/healthz`、`/readyz`、`/v1/modules`、`/v1/providers`、`/?redirect=...`）不受鉴权影响。

#### 启用鉴权

通过 **命令行 flag** 或 **环境变量** 配置 Token，两者二选一即可（flag 优先）：

```sh
# 方式 A：命令行 flag
./build/inscurascraper-server -token "my-secret-token"

# 方式 B：环境变量
export TOKEN="my-secret-token"
./build/inscurascraper-server
```

**未设置 Token（`-token` 为空）时鉴权整体关闭**，所有端点均可公开访问，适合本地开发或内网部署。生产环境务必显式配置。

Docker 场景：

```sh
docker run -d -p 8080:8080 \
  -e TOKEN=my-secret-token \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

Docker Compose 场景 —— 在 `docker-compose.yaml` 的 `environment` 段加入：

```yaml
services:
  inscurascraper:
    environment:
      TOKEN: my-secret-token
```

或通过项目根目录的 `.env` 文件加载：

```env
TOKEN=my-secret-token
```

> 💡 建议使用足够长度的随机串（如 `openssl rand -hex 32`）并通过 Secret 管理工具注入，避免明文写入仓库或镜像。

#### 调用私有端点

在请求头中附带 Token，**格式必须为 `Bearer <token>`**（大小写敏感）：

```sh
curl -H "Authorization: Bearer my-secret-token" \
  "http://localhost:8080/v1/movies/search?q=Inception"
```

校验失败一律返回：

```
HTTP/1.1 401 Unauthorized
```

```json
{ "error": { "code": 401, "message": "unauthorized" } }
```

常见原因：

- 未携带 `Authorization` 头
- 前缀不是 `Bearer`（区分大小写，不接受 `bearer`、`Token` 等）
- Token 值与服务端配置不一致

#### 更换或吊销 Token

当前实现为 **单 Token 静态配置**，更换 Token 需重启进程使新值生效；如需多 Token 管理或动态吊销，可在代码层基于 `route/auth.TokenStore` 自行扩展。

### 通用响应格式

所有端点统一返回：

```json
{
  "data": { },
  "error": { "code": 400, "message": "..." }
}
```

- 成功：只返回 `data`
- 失败：只返回 `error`，HTTP 状态码与 `error.code` 对齐

### 可选请求头

每次请求可覆盖 Provider 行为，无需重启：

| Header | 说明 |
|--------|------|
| `X-Is-Proxy` | 本次请求所有 Provider 使用的代理 URL |
| `X-Is-Api-Key-{PROVIDER}` | 覆盖指定 Provider 的 API Key（大小写不敏感） |
| `X-Is-Language` | 响应语言，BCP 47 标签，如 `zh-CN`、`en-US` |

示例：

```sh
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: zh-CN" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     "http://localhost:8080/v1/movies/search?q=Inception"
```

### 端点一览

| Method | Path | 鉴权 | 说明 |
|--------|------|------|------|
| GET | `/` | ❌ | 服务信息 |
| GET | `/healthz` | ❌ | 存活探针 |
| GET | `/readyz` | ❌ | 就绪探针（检测数据库） |
| GET | `/v1/modules` | ❌ | 构建依赖列表 |
| GET | `/v1/providers` | ❌ | 已注册 Provider 列表 |
| GET | `/v1/db/version` | ✅ | 数据库版本 |
| GET | `/v1/config/proxy` | ✅ | 当前 Provider 代理配置 |
| GET | `/v1/movies/search` | ✅ | 搜索影片 |
| GET | `/v1/movies/:provider/:id` | ✅ | 获取影片详情 |
| GET | `/v1/actors/search` | ✅ | 搜索演员 |
| GET | `/v1/actors/:provider/:id` | ✅ | 获取演员详情 |
| GET | `/v1/reviews/:provider/:id` | ✅ | 获取影片评论 |
| GET | `/?redirect=:provider:id` | ❌ | 根据 providerID 重定向到源站主页 |

### 端点详情

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
# 若数据库不可用：HTTP 503 {"status":"not_ready","error":"..."}
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

查询参数：

| 参数 | 必选 | 说明 |
|------|------|------|
| `q` | ✅ | 关键词；若传入 http(s) URL，则自动解析 Provider 和 ID 直接取详情 |
| `provider` | ❌ | 限定 Provider（忽略大小写）；不传则聚合全部 Provider |
| `fallback` | ❌ | 上游无结果时是否回退到本地 DB 缓存，默认 `true` |

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

查询参数：

| 参数 | 说明 |
|------|------|
| `lazy` | `true`（默认）= 优先读缓存；`false` = 强制回源刷新 |

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

参数与影片端点一致。演员返回示例：

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

查询参数：

| 参数 | 说明 |
|------|------|
| `homepage` | 可选；直接按 URL 抓取评论 |
| `lazy` | 同上 |

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

> 只有实现了 `MovieReviewer` 接口的 Provider 才支持此端点；否则返回 400。

#### `GET /v1/db/version`

```json
{ "data": { "version": "PostgreSQL 15.6 on x86_64-pc-linux-musl ..." } }
```

#### `GET /v1/config/proxy`

返回每个 Provider 当前的持久化代理设置（由环境变量注入，运行时只读）。

```json
{
  "data": {
    "TMDB":   "http://127.0.0.1:7890",
    "TVDB":   ""
  }
}
```

#### `GET /?redirect=TMDB:27205`

直接 `302` 重定向到该影片 / 演员在源站的主页。

### 错误响应

```json
{
  "error": {
    "code": 404,
    "message": "info not found"
  }
}
```

常见状态码：

| HTTP | 含义 |
|------|------|
| 400 | 参数错误 / ID 或 URL 格式错误 |
| 401 | 缺少或非法 Token |
| 404 | 未找到对应资源或 Provider |
| 500 | 上游抓取失败 / 数据库错误 |
| 503 | `/readyz` 数据库不可用 |

## 数据模型

完整字段定义见 [`model/movie.go`](model/movie.go) 与 [`model/actor.go`](model/actor.go)。概要：

- `MovieInfo`：`id, number, title, summary, provider, homepage, director, actors[], thumb_url, big_thumb_url, cover_url, big_cover_url, preview_video_url, preview_video_hls_url, preview_images[], maker, label, series, genres[], score, runtime, release_date`
- `MovieSearchResult`：`MovieInfo` 的轻量子集
- `ActorInfo`：`id, name, provider, homepage, summary, aliases[], images[], birthday, blood_type, cup_size, measurements, height, nationality, debut_date`
- `MovieReviewDetail`：`title, author, comment, score, date`

## 开发

### 构建 / 测试 / Lint

```sh
make              # 开发构建
make server       # 生产构建
make lint         # golangci-lint
go test ./...     # 全量单测
```

交叉编译：

```sh
make darwin-arm64 linux-amd64 windows-amd64
make releases          # 输出所有架构的 zip 到 build/
```

### 开发新 Provider

详细指南见 [CLAUDE.md](CLAUDE.md) 的 **Provider Development Guide**。简要步骤：

1. 在 `provider/<name>/` 下创建目录，嵌入 `*scraper.Scraper`
2. 实现 `provider.MovieProvider` 和/或 `ActorProvider`
3. 在 `init()` 中调用 `provider.Register(Name, New)`
4. 在 `engine/register.go` 添加 blank import

## 许可证

本项目采用 [Apache License 2.0](LICENSE) 许可。
- 许可证：[Apache 2.0](LICENSE)

## 致谢

| Library | Description |
|---------|-------------|
| [gocolly/colly](https://github.com/gocolly/colly) | Elegant scraper and crawler framework for Go |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP web framework |
| [gorm.io/gorm](https://gorm.io/) | ORM for Go |
| [robertkrimen/otto](https://github.com/robertkrimen/otto) | Pure-Go JavaScript interpreter |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | CGo-free SQLite port |
| [antchfx/xpath](https://github.com/antchfx/xpath) | XPath for HTML / XML / JSON |
| [peterbourgon/ff](https://github.com/peterbourgon/ff) | Flags / env unified parsing |
