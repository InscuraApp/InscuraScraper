# InscuraScraper

[English](README.md) | [简体中文](README.zh-CN.md) | **繁體中文** | [日本語](README.ja.md) | [한국어](README.ko.md)

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://golang.org/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?logo=github-actions)](.github/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/ghcr.io-inscuraapp%2Finscurascraper-2496ED?logo=docker)](https://github.com/orgs/InscuraApp/packages/container/package/inscurascraper)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

**InscuraScraper** 是一個以 Go 撰寫的元資料抓取 SDK 與 HTTP 服務。它透過可插拔的 Provider 機制從 TMDB、TVDB、TVmaze、AniList、Fanart.tv 等來源抓取影片與演員元資料，提供統一的 RESTful API，並使用 SQLite 或 PostgreSQL 作為本地快取。

> Forked and refactored with the original author's permission.

## 目錄

- [特色](#特色)
- [快速開始](#快速開始)
  - [二進位檔方式](#二進位檔方式)
  - [Docker 方式](#docker-方式)
  - [Docker Compose 方式](#docker-compose-方式)
- [設定](#設定)
  - [伺服器參數](#伺服器參數)
  - [Provider 設定（環境變數）](#provider-設定環境變數)
- [API 參考](#api-參考)
  - [驗證](#驗證)
  - [通用回應格式](#通用回應格式)
  - [可選請求標頭](#可選請求標頭)
  - [端點一覽](#端點一覽)
  - [端點詳情](#端點詳情)
- [資料模型](#資料模型)
- [開發](#開發)
- [貢獻 / 安全 / 授權](#貢獻--安全--授權)

## 特色

- 🔌 **可插拔 Provider 架構**：已內建 TMDB、TVDB、TVmaze、AniList、Fanart.tv，開發新來源只需實作介面並註冊
- 🚀 **RESTful API**：Gin 驅動，統一的搜尋 / 資訊 / 評論 / 代理查詢端點
- 🗄️ **雙資料庫支援**：預設記憶體 SQLite（零設定），生產環境可切換至 PostgreSQL
- ⚡ **本地快取**：先查快取再回源,降低上游配額消耗
- 🌐 **每請求客製化**：透過請求標頭動態切換代理、API Key、語言,無需重啟
- 💊 **可觀測性**：內建 `/healthz`、`/readyz` 健康檢查端點
- 🐳 **跨平台**：Linux / macOS / Windows / BSD,已提供 Dockerfile 與 Docker Compose

## 快速開始

> 💡 **開箱即用**：Docker 映像檔已發佈至 GHCR,執行 `docker pull ghcr.io/inscuraapp/inscurascraper:latest` 即可開始使用,詳見 [Docker 方式](#docker-方式)。

### 二進位檔方式

前置需求：Go 1.25+、`make`。

```sh
git clone https://github.com/InscuraApp/InscuraScraper.git
cd InscuraScraper
make                                  # 產物：build/inscurascraper-server

./build/inscurascraper-server         # 預設監聽 :8080,使用記憶體 SQLite
```

驗證：

```sh
curl -s http://localhost:8080/healthz
# {"status":"ok"}

curl -s http://localhost:8080/v1/providers | jq
```

### Docker 方式

映像檔已發佈至 **GitHub Container Registry**：`ghcr.io/inscuraapp/inscurascraper`。

**可用 tag：**

| Tag | 含義 |
|-----|------|
| `latest` | 最新的穩定版本 |
| `vX.Y.Z` | 指定版本（建議生產環境使用,例如 `v0.0.1`） |
| `X.Y` | 鎖定到次版本線（例如 `0.0`）,自動取得該次版本內的修補更新 |

**支援架構：** `linux/amd64`、`linux/arm64`

#### 拉取並執行

```sh
# 最新版本,記憶體 SQLite,無驗證
docker run --rm -p 8080:8080 \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

#### 持久化 SQLite 檔案

將資料庫檔案掛載至主機目錄,避免容器重建後資料遺失：

```sh
mkdir -p ./data

docker run -d --name inscurascraper -p 8080:8080 \
  -v $PWD/data:/data \
  -e TOKEN=change-me \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest \
  -dsn "/data/inscurascraper.db" -db-auto-migrate
```

#### 本地自行建置（可選）

若你希望從原始碼建置而非拉取預建置映像檔：

```sh
docker build -t inscurascraper:local .
docker run --rm -p 8080:8080 inscurascraper:local
```

### Docker Compose 方式

儲存庫已提供 `docker-compose.yaml`,一鍵啟動 InscuraScraper + PostgreSQL。

> **注意**：目前 `docker-compose.yaml` 預設使用本地映像檔 `inscurascraper-server:latest`。若你希望直接使用 GHCR 發佈的映像檔,將 `image:` 改為 `ghcr.io/inscuraapp/inscurascraper:latest` 即可,無需先行 `docker build`。

```sh
# 選項 1：直接使用 GHCR 映像檔（建議）
#   編輯 docker-compose.yaml,將 image: inscurascraper-server:latest
#   改為 image: ghcr.io/inscuraapp/inscurascraper:latest

# 選項 2：本地建置映像檔（需要原始碼）
docker build -t inscurascraper-server:latest .

# 啟動
docker compose up -d

# 查看記錄檔
docker compose logs -f inscurascraper
```

首次啟動會自動建立資料表（`-db-auto-migrate`）。將你的 API Token 注入 `docker-compose.yaml` 的 `environment` 段落,或透過 `.env` 檔案載入：

```env
IS_PROVIDER_TMDB__API_TOKEN=xxxxx
IS_PROVIDER_FANARTTV__API_KEY=xxxxx
IS_PROVIDER_TVDB__API_KEY=xxxxx
IS_PROVIDER_TVMAZE__API_KEY=xxxxx
```

> **注意**：`docker-compose.yaml` 會將 PostgreSQL 資料卷掛載至專案目錄的 `./db`,該目錄已經由 `.gitignore` 排除,請勿將其納入版本控制。

## 設定

### 伺服器參數

所有參數皆可透過 **命令列 flag** 或 **同名大寫環境變數** 設定（由 `peterbourgon/ff` 解析）。

| Flag | 環境變數 | 預設值 | 說明 |
|------|---------|--------|------|
| `-bind` | `BIND` | `""` | 綁定位址（留空則監聽所有網卡） |
| `-port` | `PORT` | `8080` | HTTP 連接埠 |
| `-token` | `TOKEN` | `""` | API 驗證 Token；留空則關閉驗證 |
| `-dsn` | `DSN` | `""` | 資料庫 DSN；留空則使用記憶體 SQLite |
| `-request-timeout` | `REQUEST_TIMEOUT` | `1m` | 單次上游請求逾時 |
| `-db-auto-migrate` | `DB_AUTO_MIGRATE` | `false` | 啟動時自動建立資料表（SQLite 強制開啟） |
| `-db-prepared-stmt` | `DB_PREPARED_STMT` | `false` | 啟用預先編譯陳述式 |
| `-db-max-idle-conns` | `DB_MAX_IDLE_CONNS` | `0` | 最大閒置連線 |
| `-db-max-open-conns` | `DB_MAX_OPEN_CONNS` | `0` | 最大開啟連線 |
| `-version` | `VERSION` | - | 顯示版本後結束 |

DSN 範例：

```sh
# SQLite 檔案
-dsn "/data/inscurascraper.db"

# PostgreSQL TCP
-dsn "postgres://user:pass@host:5432/inscurascraper?sslmode=disable"

# PostgreSQL Unix socket（見 docker-compose.yaml）
-dsn "postgres://user:pass@/inscurascraper?host=/var/run/postgresql"
```

### Provider 設定（環境變數）

各 Provider 的 API Key、代理、優先級等,透過帶前綴的環境變數注入：

```sh
# 同時作用於 actor 與 movie provider
IS_PROVIDER_{NAME}__{KEY}=value

# 僅作用於 actor provider
IS_ACTOR_PROVIDER_{NAME}__{KEY}=value

# 僅作用於 movie provider
IS_MOVIE_PROVIDER_{NAME}__{KEY}=value
```

常用 `{KEY}`：

| Key | 說明 |
|-----|------|
| `API_TOKEN` / `API_KEY` | 上游 API 憑證 |
| `PRIORITY` | 匹配優先級（數值越大越優先） |
| `PROXY` | HTTP/SOCKS5 代理 URL |
| `TIMEOUT` | 請求逾時（Go duration,如 `30s`） |

範例：

```sh
export IS_PROVIDER_TMDB__API_TOKEN=eyJhbGciOi...
export IS_PROVIDER_TMDB__PRIORITY=10
export IS_PROVIDER_TMDB__PROXY=http://127.0.0.1:7890
```

## API 參考

### 驗證

InscuraScraper 對 **私有端點**（參考下文 [端點一覽](#端點一覽) 中 ✅ 標記的路徑）採用簡單的 **Bearer Token** 驗證機制。公開端點（`/`、`/healthz`、`/readyz`、`/v1/modules`、`/v1/providers`、`/?redirect=...`）不受驗證影響。

#### 啟用驗證

透過 **命令列 flag** 或 **環境變數** 設定 Token,兩者二選一即可（flag 優先）：

```sh
# 方式 A：命令列 flag
./build/inscurascraper-server -token "my-secret-token"

# 方式 B：環境變數
export TOKEN="my-secret-token"
./build/inscurascraper-server
```

**未設定 Token（`-token` 為空）時驗證整體關閉**,所有端點皆可公開存取,適合本地開發或內網部署。生產環境請務必明確設定。

Docker 情境：

```sh
docker run -d -p 8080:8080 \
  -e TOKEN=my-secret-token \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

Docker Compose 情境 —— 於 `docker-compose.yaml` 的 `environment` 段落加入：

```yaml
services:
  inscurascraper:
    environment:
      TOKEN: my-secret-token
```

或透過專案根目錄的 `.env` 檔案載入：

```env
TOKEN=my-secret-token
```

> 💡 建議使用足夠長度的隨機字串（如 `openssl rand -hex 32`）並透過機密管理工具注入,避免明文寫入儲存庫或映像檔。

#### 呼叫私有端點

於請求標頭中附上 Token,**格式必須為 `Bearer <token>`**（區分大小寫）：

```sh
curl -H "Authorization: Bearer my-secret-token" \
  "http://localhost:8080/v1/movies/search?q=Inception"
```

驗證失敗一律回傳：

```
HTTP/1.1 401 Unauthorized
```

```json
{ "error": { "code": 401, "message": "unauthorized" } }
```

常見原因：

- 未附帶 `Authorization` 標頭
- 前綴不是 `Bearer`（區分大小寫,不接受 `bearer`、`Token` 等）
- Token 值與伺服器端設定不一致

#### 更換或撤銷 Token

目前實作為 **單一 Token 靜態設定**,更換 Token 需重新啟動程序使新值生效；若需多 Token 管理或動態撤銷,可在程式碼層基於 `route/auth.TokenStore` 自行擴充。

### 通用回應格式

所有端點統一回傳：

```json
{
  "data": { },
  "error": { "code": 400, "message": "..." }
}
```

- 成功：僅回傳 `data`
- 失敗：僅回傳 `error`,HTTP 狀態碼與 `error.code` 對齊

### 可選請求標頭

每次請求可覆蓋 Provider 行為,無需重啟：

| Header | 說明 |
|--------|------|
| `X-Is-Proxy` | 本次請求所有 Provider 使用的代理 URL |
| `X-Is-Api-Key-{PROVIDER}` | 覆蓋指定 Provider 的 API Key（不區分大小寫） |
| `X-Is-Language` | 回應語言,BCP 47 標籤,如 `zh-TW`、`en-US` |

範例：

```sh
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: zh-TW" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     "http://localhost:8080/v1/movies/search?q=Inception"
```

### 端點一覽

| Method | Path | 驗證 | 說明 |
|--------|------|------|------|
| GET | `/` | ❌ | 服務資訊 |
| GET | `/healthz` | ❌ | 存活探針 |
| GET | `/readyz` | ❌ | 就緒探針（檢測資料庫） |
| GET | `/v1/modules` | ❌ | 建置相依清單 |
| GET | `/v1/providers` | ❌ | 已註冊 Provider 清單 |
| GET | `/v1/db/version` | ✅ | 資料庫版本 |
| GET | `/v1/config/proxy` | ✅ | 當前 Provider 代理設定 |
| GET | `/v1/movies/search` | ✅ | 搜尋影片 |
| GET | `/v1/movies/:provider/:id` | ✅ | 取得影片詳情 |
| GET | `/v1/actors/search` | ✅ | 搜尋演員 |
| GET | `/v1/actors/:provider/:id` | ✅ | 取得演員詳情 |
| GET | `/v1/reviews/:provider/:id` | ✅ | 取得影片評論 |
| GET | `/?redirect=:provider:id` | ❌ | 根據 providerID 重新導向至源站首頁 |

### 端點詳情

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
# 若資料庫不可用：HTTP 503 {"status":"not_ready","error":"..."}
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

查詢參數：

| 參數 | 必填 | 說明 |
|------|------|------|
| `q` | ✅ | 關鍵字；若傳入 http(s) URL,則自動解析 Provider 和 ID 直接取詳情 |
| `provider` | ❌ | 限定 Provider（忽略大小寫）；不傳則聚合全部 Provider |
| `fallback` | ❌ | 上游無結果時是否回退到本地 DB 快取,預設 `true` |

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

查詢參數：

| 參數 | 說明 |
|------|------|
| `lazy` | `true`（預設）= 優先讀快取；`false` = 強制回源重新抓取 |

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

參數與影片端點一致。演員回傳範例：

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

查詢參數：

| 參數 | 說明 |
|------|------|
| `homepage` | 可選；直接依 URL 抓取評論 |
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

> 僅實作了 `MovieReviewer` 介面的 Provider 才支援此端點；否則回傳 400。

#### `GET /v1/db/version`

```json
{ "data": { "version": "PostgreSQL 15.6 on x86_64-pc-linux-musl ..." } }
```

#### `GET /v1/config/proxy`

回傳每個 Provider 目前的持久化代理設定（由環境變數注入,執行期間唯讀）。

```json
{
  "data": {
    "TMDB":   "http://127.0.0.1:7890",
    "TVDB":   ""
  }
}
```

#### `GET /?redirect=TMDB:27205`

直接以 `302` 重新導向至該影片 / 演員在源站的首頁。

### 錯誤回應

```json
{
  "error": {
    "code": 404,
    "message": "info not found"
  }
}
```

常見狀態碼：

| HTTP | 含義 |
|------|------|
| 400 | 參數錯誤 / ID 或 URL 格式錯誤 |
| 401 | 缺少或非法 Token |
| 404 | 未找到對應資源或 Provider |
| 500 | 上游抓取失敗 / 資料庫錯誤 |
| 503 | `/readyz` 資料庫不可用 |

## 資料模型

完整欄位定義見 [`model/movie.go`](model/movie.go) 與 [`model/actor.go`](model/actor.go)。概要：

- `MovieInfo`：`id, number, title, summary, provider, homepage, director, actors[], thumb_url, big_thumb_url, cover_url, big_cover_url, preview_video_url, preview_video_hls_url, preview_images[], maker, label, series, genres[], score, runtime, release_date`
- `MovieSearchResult`：`MovieInfo` 的輕量子集
- `ActorInfo`：`id, name, provider, homepage, summary, aliases[], images[], birthday, blood_type, cup_size, measurements, height, nationality, debut_date`
- `MovieReviewDetail`：`title, author, comment, score, date`

## 開發

### 建置 / 測試 / Lint

```sh
make              # 開發建置
make server       # 生產建置
make lint         # golangci-lint
go test ./...     # 全量單元測試
```

交叉編譯：

```sh
make darwin-arm64 linux-amd64 windows-amd64
make releases          # 輸出所有架構的 zip 至 build/
```

### 開發新 Provider

詳細指南見 [CLAUDE.md](CLAUDE.md) 的 **Provider Development Guide** 與 [CONTRIBUTING.md](CONTRIBUTING.md)。簡要步驟：

1. 於 `provider/<name>/` 下建立目錄,嵌入 `*scraper.Scraper`
2. 實作 `provider.MovieProvider` 和/或 `ActorProvider`
3. 於 `init()` 中呼叫 `provider.Register(Name, New)`
4. 於 `engine/register.go` 加入 blank import

## 貢獻 / 安全 / 授權

- [貢獻指南](CONTRIBUTING.md)
- [行為準則](CODE_OF_CONDUCT.md)
- [安全揭露](SECURITY.md)（請勿公開提 Issue）
- [變更日誌](CHANGELOG.md)
- 授權：[Apache 2.0](LICENSE)

## 致謝

| Library | Description |
|---------|-------------|
| [gocolly/colly](https://github.com/gocolly/colly) | Elegant scraper and crawler framework for Go |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP web framework |
| [gorm.io/gorm](https://gorm.io/) | ORM for Go |
| [robertkrimen/otto](https://github.com/robertkrimen/otto) | Pure-Go JavaScript interpreter |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | CGo-free SQLite port |
| [antchfx/xpath](https://github.com/antchfx/xpath) | XPath for HTML / XML / JSON |
| [peterbourgon/ff](https://github.com/peterbourgon/ff) | Flags / env unified parsing |
