# InscuraScraper

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | **日本語** | [한국어](README.ko.md)

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://golang.org/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?logo=github-actions)](.github/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/ghcr.io-inscuraapp%2Finscurascraper-2496ED?logo=docker)](https://github.com/orgs/InscuraApp/packages/container/package/inscurascraper)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

**InscuraScraper** は Go で書かれたメタデータスクレイピング SDK と HTTP サービスです。プラガブルな Provider アーキテクチャにより TMDB、TVDB、TVmaze、AniList、Fanart.tv などのソースから映画・俳優メタデータを取得し、統一された RESTful API を提供し、SQLite または PostgreSQL をローカルキャッシュとして使用します。

> Forked and refactored with the original author's permission.

## 目次

- [特徴](#特徴)
- [クイックスタート](#クイックスタート)
  - [バイナリ](#バイナリ)
  - [Docker](#docker)
  - [Docker Compose](#docker-compose)
- [設定](#設定)
  - [サーバーオプション](#サーバーオプション)
  - [Provider 設定（環境変数）](#provider-設定環境変数)
- [API リファレンス](#api-リファレンス)
  - [認証](#認証)
  - [共通レスポンス形式](#共通レスポンス形式)
  - [オプションのリクエストヘッダー](#オプションのリクエストヘッダー)
  - [エンドポイント一覧](#エンドポイント一覧)
  - [エンドポイント詳細](#エンドポイント詳細)
- [データモデル](#データモデル)
- [開発](#開発)
- [コントリビューション / セキュリティ / ライセンス](#コントリビューション--セキュリティ--ライセンス)

## 特徴

- 🔌 **プラガブルな Provider アーキテクチャ**：TMDB、TVDB、TVmaze、AniList、Fanart.tv を内蔵。新しいソースの追加はインターフェースを実装して登録するだけ
- 🚀 **RESTful API**：Gin ベースで、検索・情報取得・レビュー・プロキシ設定の統一エンドポイントを提供
- 🗄️ **デュアルデータベース対応**：デフォルトはインメモリ SQLite（設定不要）、本番では PostgreSQL に切り替え可能
- ⚡ **ローカルキャッシュ**：まずキャッシュを参照し、見つからなければ上流にフォールバック。API クォータを節約
- 🌐 **リクエストごとのカスタマイズ**：リクエストヘッダーでプロキシ・API キー・言語を動的に切り替え可能（再起動不要）
- 💊 **オブザーバビリティ**：`/healthz`、`/readyz` ヘルスチェックエンドポイントを内蔵
- 🐳 **マルチプラットフォーム**：Linux / macOS / Windows / BSD 対応、Dockerfile と Docker Compose を同梱

## クイックスタート

> 💡 **すぐ使える**：Docker イメージは GHCR に公開済み。`docker pull ghcr.io/inscuraapp/inscurascraper:latest` を実行するだけで開始できます。詳細は [Docker](#docker) を参照。

### バイナリ

前提：Go 1.25+、`make`。

```sh
git clone https://github.com/InscuraApp/InscuraScraper.git
cd InscuraScraper
make                                  # 成果物：build/inscurascraper-server

./build/inscurascraper-server         # デフォルトで :8080 を待ち受け、インメモリ SQLite を使用
```

動作確認：

```sh
curl -s http://localhost:8080/healthz
# {"status":"ok"}

curl -s http://localhost:8080/v1/providers | jq
```

### Docker

イメージは **GitHub Container Registry** に公開されています：`ghcr.io/inscuraapp/inscurascraper`。

**利用可能なタグ：**

| タグ | 意味 |
|------|------|
| `latest` | 最新の安定版 |
| `vX.Y.Z` | バージョン指定（本番環境推奨、例：`v0.0.1`） |
| `X.Y` | マイナーバージョンをロック（例：`0.0`）、パッチ更新を自動取得 |

**対応アーキテクチャ：** `linux/amd64`、`linux/arm64`

#### プル & 実行

```sh
# 最新版、インメモリ SQLite、認証なし
docker run --rm -p 8080:8080 \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

#### SQLite ファイルを永続化

データベースファイルをホスト側にマウントし、コンテナ再作成後もデータが失われないようにします：

```sh
mkdir -p ./data

docker run -d --name inscurascraper -p 8080:8080 \
  -v $PWD/data:/data \
  -e TOKEN=change-me \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest \
  -dsn "/data/inscurascraper.db" -db-auto-migrate
```

#### ローカルビルド（オプション）

ビルド済みイメージのプルではなく、ソースからビルドしたい場合：

```sh
docker build -t inscurascraper:local .
docker run --rm -p 8080:8080 inscurascraper:local
```

### Docker Compose

リポジトリには `docker-compose.yaml` が同梱されており、ワンコマンドで InscuraScraper + PostgreSQL を起動できます。

> **注意**：現在の `docker-compose.yaml` はデフォルトでローカルイメージ `inscurascraper-server:latest` を使用します。GHCR に公開されているイメージを直接使う場合は、`image:` を `ghcr.io/inscuraapp/inscurascraper:latest` に変更するだけで、`docker build` は不要です。

```sh
# 選択肢 1：GHCR イメージを使用（推奨）
#   docker-compose.yaml を編集し、image: inscurascraper-server:latest を
#   image: ghcr.io/inscuraapp/inscurascraper:latest に変更

# 選択肢 2：ローカルビルド（ソースが必要）
docker build -t inscurascraper-server:latest .

# 起動
docker compose up -d

# ログ追跡
docker compose logs -f inscurascraper
```

初回起動時にテーブルが自動作成されます（`-db-auto-migrate`）。`docker-compose.yaml` の `environment` セクション、または `.env` ファイル経由で API トークンを注入してください：

```env
IS_PROVIDER_TMDB__API_TOKEN=xxxxx
IS_PROVIDER_FANARTTV__API_KEY=xxxxx
IS_PROVIDER_TVDB__API_KEY=xxxxx
IS_PROVIDER_TVMAZE__API_KEY=xxxxx
```

> **注意**：`docker-compose.yaml` はプロジェクト直下の `./db` に PostgreSQL データボリュームをマウントします。このディレクトリは `.gitignore` に追加済みのため、バージョン管理に含めないでください。

## 設定

### サーバーオプション

すべてのオプションは **コマンドラインフラグ** または **同名の大文字環境変数** で設定できます（`peterbourgon/ff` が解析）。

| Flag | 環境変数 | デフォルト | 説明 |
|------|---------|-----------|------|
| `-bind` | `BIND` | `""` | バインドアドレス（空で全インターフェースを待ち受け） |
| `-port` | `PORT` | `8080` | HTTP ポート |
| `-token` | `TOKEN` | `""` | API 認証トークン。空で認証を無効化 |
| `-dsn` | `DSN` | `""` | データベース DSN。空でインメモリ SQLite |
| `-request-timeout` | `REQUEST_TIMEOUT` | `1m` | 上流リクエストごとのタイムアウト |
| `-db-auto-migrate` | `DB_AUTO_MIGRATE` | `false` | 起動時にテーブル自動作成（SQLite では強制 ON） |
| `-db-prepared-stmt` | `DB_PREPARED_STMT` | `false` | プリペアドステートメントを有効化 |
| `-db-max-idle-conns` | `DB_MAX_IDLE_CONNS` | `0` | DB の最大アイドルコネクション |
| `-db-max-open-conns` | `DB_MAX_OPEN_CONNS` | `0` | DB の最大オープンコネクション |
| `-version` | `VERSION` | - | バージョンを表示して終了 |

DSN の例：

```sh
# SQLite ファイル
-dsn "/data/inscurascraper.db"

# PostgreSQL TCP
-dsn "postgres://user:pass@host:5432/inscurascraper?sslmode=disable"

# PostgreSQL Unix socket（docker-compose.yaml 参照）
-dsn "postgres://user:pass@/inscurascraper?host=/var/run/postgresql"
```

### Provider 設定（環境変数）

各 Provider の API キー、プロキシ、優先度などはプレフィックス付き環境変数で注入します：

```sh
# actor と movie の両方の provider に適用
IS_PROVIDER_{NAME}__{KEY}=value

# actor provider のみ
IS_ACTOR_PROVIDER_{NAME}__{KEY}=value

# movie provider のみ
IS_MOVIE_PROVIDER_{NAME}__{KEY}=value
```

主な `{KEY}`：

| Key | 説明 |
|-----|------|
| `API_TOKEN` / `API_KEY` | 上流 API の認証情報 |
| `PRIORITY` | マッチング優先度（大きいほど優先） |
| `PROXY` | HTTP/SOCKS5 プロキシ URL |
| `TIMEOUT` | リクエストタイムアウト（Go duration、例：`30s`） |

例：

```sh
export IS_PROVIDER_TMDB__API_TOKEN=eyJhbGciOi...
export IS_PROVIDER_TMDB__PRIORITY=10
export IS_PROVIDER_TMDB__PROXY=http://127.0.0.1:7890
```

## API リファレンス

### 認証

InscuraScraper は **プライベートエンドポイント**（下記 [エンドポイント一覧](#エンドポイント一覧) で ✅ マークされたパス）にシンプルな **Bearer Token** 認証を採用します。公開エンドポイント（`/`、`/healthz`、`/readyz`、`/v1/modules`、`/v1/providers`、`/?redirect=...`）は認証の影響を受けません。

#### 認証を有効化

**コマンドラインフラグ** または **環境変数** のいずれかで Token を設定します（フラグが優先）：

```sh
# 方法 A：コマンドラインフラグ
./build/inscurascraper-server -token "my-secret-token"

# 方法 B：環境変数
export TOKEN="my-secret-token"
./build/inscurascraper-server
```

**Token が未設定（`-token` が空）の場合、認証は完全に無効化され**、すべてのエンドポイントが公開されます。ローカル開発や内部ネットワーク向けなら問題ありませんが、本番環境では必ず明示的に設定してください。

Docker の場合：

```sh
docker run -d -p 8080:8080 \
  -e TOKEN=my-secret-token \
  -e IS_PROVIDER_TMDB__API_TOKEN=<your-tmdb-token> \
  ghcr.io/inscuraapp/inscurascraper:latest
```

Docker Compose の場合 —— `docker-compose.yaml` の `environment` セクションに追加：

```yaml
services:
  inscurascraper:
    environment:
      TOKEN: my-secret-token
```

またはリポジトリ直下の `.env` ファイル経由で読み込み：

```env
TOKEN=my-secret-token
```

> 💡 十分な長さのランダム文字列（例：`openssl rand -hex 32`）をシークレット管理ツール経由で注入することを推奨します。リポジトリやイメージに平文で書き込まないでください。

#### プライベートエンドポイントの呼び出し

リクエストヘッダーに Token を付与します。**形式は必ず `Bearer <token>`**（大文字小文字を区別）：

```sh
curl -H "Authorization: Bearer my-secret-token" \
  "http://localhost:8080/v1/movies/search?q=Inception"
```

検証失敗時は常に次を返します：

```
HTTP/1.1 401 Unauthorized
```

```json
{ "error": { "code": 401, "message": "unauthorized" } }
```

よくある原因：

- `Authorization` ヘッダー未付与
- プレフィックスが `Bearer` でない（大文字小文字を区別。`bearer`、`Token` などは拒否）
- Token の値がサーバー設定と不一致

#### Token のローテーション / 失効

現在の実装は **シングル Token の静的設定** です。Token の変更にはプロセス再起動が必要です。複数 Token 管理や動的失効が必要な場合は、`route/auth.TokenStore` をベースにコード側で拡張してください。

### 共通レスポンス形式

すべてのエンドポイントは次の形式で返します：

```json
{
  "data": { },
  "error": { "code": 400, "message": "..." }
}
```

- 成功：`data` のみ
- 失敗：`error` のみ。HTTP ステータスコードは `error.code` と一致

### オプションのリクエストヘッダー

リクエストごとに Provider の挙動を上書きできます（再起動不要）：

| Header | 説明 |
|--------|------|
| `X-Is-Proxy` | このリクエストで全 Provider が使うプロキシ URL |
| `X-Is-Api-Key-{PROVIDER}` | 指定 Provider の API キーを上書き（大文字小文字を区別しない） |
| `X-Is-Language` | レスポンス言語。BCP 47 タグ、例：`ja-JP`、`en-US` |

例：

```sh
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Is-Language: ja-JP" \
     -H "X-Is-Api-Key-TMDB: eyJhbGciOi..." \
     "http://localhost:8080/v1/movies/search?q=Inception"
```

### エンドポイント一覧

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/` | ❌ | サービス情報 |
| GET | `/healthz` | ❌ | Liveness プローブ |
| GET | `/readyz` | ❌ | Readiness プローブ（DB を検査） |
| GET | `/v1/modules` | ❌ | ビルド依存関係リスト |
| GET | `/v1/providers` | ❌ | 登録済み Provider 一覧 |
| GET | `/v1/db/version` | ✅ | データベースバージョン |
| GET | `/v1/config/proxy` | ✅ | 現在の Provider プロキシ設定 |
| GET | `/v1/movies/search` | ✅ | 映画検索 |
| GET | `/v1/movies/:provider/:id` | ✅ | 映画詳細取得 |
| GET | `/v1/actors/search` | ✅ | 俳優検索 |
| GET | `/v1/actors/:provider/:id` | ✅ | 俳優詳細取得 |
| GET | `/v1/reviews/:provider/:id` | ✅ | 映画レビュー取得 |
| GET | `/?redirect=:provider:id` | ❌ | providerID に基づきソースサイトのホームへリダイレクト |

### エンドポイント詳細

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
# DB 到達不可の場合：HTTP 503 {"status":"not_ready","error":"..."}
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

クエリパラメータ：

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `q` | ✅ | キーワード。http(s) URL を渡すと Provider と ID を自動解析して詳細を取得 |
| `provider` | ❌ | Provider を限定（大文字小文字を無視）。未指定なら全 Provider を集約 |
| `fallback` | ❌ | 上流に結果がない場合にローカル DB キャッシュへフォールバックするか。デフォルト `true` |

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

クエリパラメータ：

| パラメータ | 説明 |
|-----------|------|
| `lazy` | `true`（デフォルト）= キャッシュを優先；`false` = 強制的に上流から再取得 |

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

パラメータは映画エンドポイントと同じ。俳優レスポンスの例：

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

クエリパラメータ：

| パラメータ | 説明 |
|-----------|------|
| `homepage` | 任意。URL を直接指定してレビューを取得 |
| `lazy` | 上記と同じ |

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

> `MovieReviewer` インターフェースを実装している Provider のみ対応。それ以外は 400 を返します。

#### `GET /v1/db/version`

```json
{ "data": { "version": "PostgreSQL 15.6 on x86_64-pc-linux-musl ..." } }
```

#### `GET /v1/config/proxy`

各 Provider の現在の永続化プロキシ設定を返します（環境変数経由で注入、実行時は読み取り専用）。

```json
{
  "data": {
    "TMDB":   "http://127.0.0.1:7890",
    "TVDB":   ""
  }
}
```

#### `GET /?redirect=TMDB:27205`

該当の映画 / 俳優のソースサイトホームページに `302` リダイレクトします。

### エラーレスポンス

```json
{
  "error": {
    "code": 404,
    "message": "info not found"
  }
}
```

主なステータスコード：

| HTTP | 意味 |
|------|------|
| 400 | パラメータ不正 / ID または URL 形式エラー |
| 401 | Token 不足または不正 |
| 404 | 対応リソースまたは Provider が見つからない |
| 500 | 上流スクレイピング失敗 / データベースエラー |
| 503 | `/readyz` でデータベース到達不可 |

## データモデル

完全なフィールド定義は [`model/movie.go`](model/movie.go) と [`model/actor.go`](model/actor.go) を参照。概要：

- `MovieInfo`：`id, number, title, summary, provider, homepage, director, actors[], thumb_url, big_thumb_url, cover_url, big_cover_url, preview_video_url, preview_video_hls_url, preview_images[], maker, label, series, genres[], score, runtime, release_date`
- `MovieSearchResult`：`MovieInfo` の軽量サブセット
- `ActorInfo`：`id, name, provider, homepage, summary, aliases[], images[], birthday, blood_type, cup_size, measurements, height, nationality, debut_date`
- `MovieReviewDetail`：`title, author, comment, score, date`

## 開発

### ビルド / テスト / Lint

```sh
make              # 開発ビルド
make server       # 本番ビルド
make lint         # golangci-lint
go test ./...     # フルテスト
```

クロスコンパイル：

```sh
make darwin-arm64 linux-amd64 windows-amd64
make releases          # 全アーキテクチャの zip を build/ に出力
```

### 新しい Provider の開発

詳細ガイドは [CLAUDE.md](CLAUDE.md) の **Provider Development Guide** と [CONTRIBUTING.md](CONTRIBUTING.md) を参照。概要：

1. `provider/<name>/` にディレクトリを作成し、`*scraper.Scraper` を埋め込む
2. `provider.MovieProvider` および/または `ActorProvider` を実装
3. `init()` で `provider.Register(Name, New)` を呼び出す
4. `engine/register.go` に blank import を追加

## コントリビューション / セキュリティ / ライセンス

- [コントリビューションガイド](CONTRIBUTING.md)
- [行動規範](CODE_OF_CONDUCT.md)
- [セキュリティポリシー](SECURITY.md)（公開 Issue を立てないでください）
- [変更履歴](CHANGELOG.md)
- ライセンス：[Apache 2.0](LICENSE)

## 謝辞

| Library | Description |
|---------|-------------|
| [gocolly/colly](https://github.com/gocolly/colly) | Elegant scraper and crawler framework for Go |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP web framework |
| [gorm.io/gorm](https://gorm.io/) | ORM for Go |
| [robertkrimen/otto](https://github.com/robertkrimen/otto) | Pure-Go JavaScript interpreter |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | CGo-free SQLite port |
| [antchfx/xpath](https://github.com/antchfx/xpath) | XPath for HTML / XML / JSON |
| [peterbourgon/ff](https://github.com/peterbourgon/ff) | Flags / env unified parsing |
