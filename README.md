# 🔗 URL Shortener (Go)

![Go](https://img.shields.io/badge/Go_1.22-00ADD8?logo=go&logoColor=white)
![REST](https://img.shields.io/badge/REST-API-1F6FEB)
![Docker](https://img.shields.io/badge/Docker-distroless-2496ED?logo=docker&logoColor=white)
![CI](https://github.com/asad110abbas/url-shortener-go/actions/workflows/ci.yml/badge.svg)

A small, production-minded **URL shortener** written in Go with **zero third-party dependencies**. It uses the
Go 1.22 pattern-matching `net/http` router, structured logging, graceful shutdown and race-tested concurrency.

## Features
- Random 7-character codes from an unambiguous alphabet (no `0/O`, `1/l/I`), generated with `crypto/rand`
- Custom aliases with validation and conflict detection
- Optional link expiry (`410 Gone` after TTL)
- Click counting and last-click timestamp
- Concurrency-safe store (`sync.RWMutex`) with **atomic JSON-file persistence**
- Errors mapped to proper HTTP status codes (400 / 404 / 409 / 410)
- Request logging middleware (`log/slog`), graceful shutdown on SIGTERM
- Tiny distroless Docker image

## API
| Method | Path | Body / result |
|---|---|---|
| `POST` | `/api/links` | `{"url": "...", "custom_code": "promo", "ttl_hours": 24}` → `201` with `short_url` |
| `GET` | `/{code}` | `302` redirect to the target |
| `GET` | `/api/links/{code}` | Link details + click stats |
| `DELETE` | `/api/links/{code}` | `204` |
| `GET` | `/healthz` | `{"status":"ok"}` |

## Run
```bash
go run ./cmd/server                    # :8080
go test -race -cover ./...

curl -s -X POST localhost:8080/api/links -d '{"url":"https://go.dev/doc","custom_code":"godoc"}'
curl -i localhost:8080/godoc           # 302 -> https://go.dev/doc
```
Config via env: `ADDR`, `BASE_URL`, `DATA_FILE`.

## Layout
```
cmd/server/          entrypoint, config, graceful shutdown
internal/shortener/  domain logic + persistence (unit tests, concurrency test)
internal/httpapi/    HTTP handlers, error mapping, logging middleware (httptest tests)
```
