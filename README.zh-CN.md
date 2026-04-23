# InscuraScraper

[English](README.md) | **简体中文**

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://golang.org/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?logo=github-actions)](.github/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/ghcr.io-inscuraapp%2Finscurascraper-2496ED?logo=docker)](https://github.com/orgs/InscuraApp/packages/container/package/inscurascraper)

**InscuraScraper** 是一个用 Go 编写的元数据抓取 SDK 与 HTTP 服务。它通过可插拔的 Provider 机制从 TMDB、TVDB、TVmaze、AniList、Fanart.tv、Trakt 等源抓取影片与演员元数据，提供统一的 RESTful API，并使用 SQLite 或 PostgreSQL 作为本地缓存。

> Forked and refactored with the original author's permission.

## 目录

- [特性](#特性)
- [快速开始](#快速开始)
  - [二进制方式](#二进制方式)
  - [Docker 方式](#docker-方式)
  - [Docker Compose 方式](#docker-compose-方式)
- [配置](#配置)
  - [服务器参数](#服务器参数)
  - [Provider 环境变量](#provider-环境变量)
  - [每请求覆盖请求头](#每请求覆盖请求头)
- [API 参考](#api-参考)
  - [鉴权](#鉴权)
  - [通用响应格式](#通用响应格式)
  - [Query 参数参考](#query-参数参考)
  - [端点一览](#端点一览)
  - [端点详情](#端点详情)
- [数据模型](#数据模型)
- [开发](#开发)
- [贡献 / 安全 / 许可证](#贡献--安全--许可证)

## 特性

- 🔌 **可插拔 Provider 架构**：已内置 TMDB、TVDB、TVmaze、AniList、Fanart.tv、Trakt，开发新源只需实现接口并注册
- 🚀 **RESTful API**：Gin 驱动，统一的搜索 / 信息 / 评论 / 代理查询端点
- 🗄️ **双数据库支持**：默认内存 SQLite（零配置），生产可切到 PostgreSQL
- ⚡ **本地缓存**：先查缓存再回源，降低上游配额消耗
- 🌐 **每请求定制**：通过请求头动态切换代理、API Key、语言，无需重启
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
IS_PROVIDER_TRAKT__CLIENT_ID=xxxxx
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

### Provider 环境变量

每个 Provider 的 API Key、代理、优先级等在启动时通过带前缀的环境变量注入，属于**全局配置**。如需单请求级别的覆盖，请使用[每请求覆盖请求头](#每请求覆盖请求头)。

#### 变量命名规则

```
IS_PROVIDER_{NAME}__{KEY}=value          # 同时作用于 actor 和 movie 子 provider
IS_ACTOR_PROVIDER_{NAME}__{KEY}=value    # 仅作用于 actor 子 provider
IS_MOVIE_PROVIDER_{NAME}__{KEY}=value    # 仅作用于 movie 子 provider
```

`{NAME}` 为 Provider 名称的**大写形式**（如 `TMDB`、`TVDB`、`TRAKT`）。  
`{KEY}` 为下表中的键名。

#### 通用键（适用于任意 Provider）

| Key | 类型 | 说明 |
|-----|------|------|
| `PRIORITY` | float | 匹配优先级，数值越大越优先，多源结果合并时决定排序 |
| `PROXY` | string | HTTP 或 SOCKS5 代理 URL，如 `http://127.0.0.1:7890` 或 `socks5://127.0.0.1:1080` |
| `TIMEOUT` | duration | 单次上游请求超时，Go duration 格式，如 `30s`、`2m` |

#### 内置 Provider 凭证键

| Provider | 环境变量 | 是否必填 | 获取地址 |
|----------|---------|---------|---------|
| **TMDB** | `IS_PROVIDER_TMDB__API_TOKEN` | 是 | [themoviedb.org/settings/api](https://www.themoviedb.org/settings/api) — 使用 **Bearer Token（v4 鉴权）** |
| **TVDB** | `IS_PROVIDER_TVDB__API_KEY` | 是 | [thetvdb.com/api-information](https://thetvdb.com/api-information) |
| **Fanart.tv** | `IS_PROVIDER_FANARTTV__API_KEY` | 是 | [fanart.tv/get-an-api-key](https://fanart.tv/get-an-api-key/) |
| **Trakt** | `IS_PROVIDER_TRAKT__CLIENT_ID` | 是 | [trakt.tv/oauth/applications](https://trakt.tv/oauth/applications) — 注册应用后复制 **Client ID** |
| **AniList** | *(无)* | — | 公开 API，无需 Key |
| **TVmaze** | *(无)* | — | 公开 API，无需 Key |
| **Bangumi** | *(无)* | — | 公开 API，无需 Key |

#### 完整示例

```sh
# 凭证
export IS_PROVIDER_TMDB__API_TOKEN=eyJhbGciOiJSUzI1NiJ9...
export IS_PROVIDER_TVDB__API_KEY=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
export IS_PROVIDER_FANARTTV__API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export IS_PROVIDER_TRAKT__CLIENT_ID=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 优先级调整（数值越大越优先）
export IS_PROVIDER_TMDB__PRIORITY=1000
export IS_PROVIDER_TRAKT__PRIORITY=900

# 按 Provider 配置代理（覆盖全局代理）
export IS_PROVIDER_TMDB__PROXY=http://127.0.0.1:7890
export IS_PROVIDER_TVDB__PROXY=socks5://127.0.0.1:1080

# 按 Provider 配置超时
export IS_PROVIDER_TMDB__TIMEOUT=30s
export IS_PROVIDER_TRAKT__TIMEOUT=20s

# 仅针对 TMDB 的 movie 子 provider 调整优先级
export IS_MOVIE_PROVIDER_TMDB__PRIORITY=1100
```

### 每请求覆盖请求头

通过以下请求头，可以在**单次请求**中覆盖代理、API Key 或响应语言，无需重启服务。适用于多租户场景或临时切换 Key。

| Header | 说明 |
|--------|------|
| `X-Is-Proxy` | 本次请求所有 Provider 使用的代理 URL（覆盖全局 env 代理） |
| `X-Is-Api-Key-{PROVIDER}` | 覆盖指定 Provider 的 API Key（大小写不敏感；Provider 名称与 `/v1/providers` 返回的一致） |
| `X-Is-Language` | 响应语言，BCP 47 标签，如 `zh-CN`、`en-US`、`ja-JP` |

> **优先级**：请求头 > 全局 `IS_PROVIDER_*` 环境变量。

#### 示例

```sh
# 本次请求使用不同的 TMDB Token
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     "http://localhost:8080/v1/movies/search?q=Inception"

# 本次请求全部上游通过指定代理访问
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Proxy: socks5://127.0.0.1:1080" \
     "http://localhost:8080/v1/movies/TMDB/27205"

# 请求中文元数据
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: zh-CN" \
     "http://localhost:8080/v1/movies/search?q=盗梦空间"

# 组合使用：代理 + 语言 + 多个 Provider Key
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Proxy: http://127.0.0.1:7890" \
     -H "X-Is-Language: ja-JP" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     -H "X-Is-Api-Key-Trakt: your-trakt-client-id" \
     "http://localhost:8080/v1/movies/search?q=Inception"
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

### Query 参数参考

所有端点所支持的 Query 参数汇总。

#### 搜索端点（`/v1/movies/search`、`/v1/actors/search`）

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `q` | string | **必填** | 搜索关键词。若传入 `http(s)://` 格式的 URL，则自动解析 Provider 和 ID，直接返回详情而非搜索结果 |
| `provider` | string | *(全部)* | 限定到单个 Provider（大小写不敏感，如 `TMDB`、`Trakt`）。不传则聚合并去重所有已注册 Provider 的结果 |
| `fallback` | bool | `true` | 上游无结果时是否回退到本地 DB 缓存 |

```sh
# 跨全部 Provider 的关键词搜索
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/search?q=Inception"

# 只从 TMDB 搜索
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/search?q=Inception&provider=TMDB"

# 不回退缓存（仅上游）
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/search?q=Inception&fallback=false"

# 传入完整 URL，触发详情获取而非文本搜索
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/search?q=https://www.themoviedb.org/movie/27205"
```

#### 详情端点（`/v1/movies/:provider/:id`、`/v1/actors/:provider/:id`）

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `lazy` | bool | `true` | `true` = 优先读本地缓存，无缓存才回源；`false` = 强制回源拉取最新数据并更新缓存 |

```sh
# 读缓存（默认）
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/TMDB/27205"

# 强制回源刷新
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/TMDB/27205?lazy=false"

# Trakt 使用 slug 作为 ID
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/movies/Trakt/inception"

curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/actors/Trakt/leonardo-dicaprio"
```

#### 评论端点（`/v1/reviews/:provider/:id`）

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `homepage` | string | *(无)* | 直接按此 URL 抓取评论，而不使用路径中的 `id` |
| `lazy` | bool | `true` | 与详情端点相同的缓存语义 |

```sh
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/reviews/TMDB/27205"

# 强制刷新
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/reviews/TMDB/27205?lazy=false"
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
      "TMDB":  "https://www.themoviedb.org",
      "TVDB":  "https://thetvdb.com",
      "Trakt": "https://trakt.tv"
    },
    "movie_providers": {
      "TMDB":   "https://www.themoviedb.org",
      "TVmaze": "https://www.tvmaze.com",
      "Trakt":  "https://trakt.tv"
    }
  }
}
```

#### `GET /v1/movies/search`

详细参数说明见 [Query 参数参考 — 搜索端点](#搜索端点v1moviessearchv1actorssearch)。

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

详细参数说明见 [Query 参数参考 — 详情端点](#详情端点v1moviesprovideridv1actorsproviderid)。

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

详细参数说明见 [Query 参数参考 — 评论端点](#评论端点v1reviewsproviderid)。

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
