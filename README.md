Simple demo of concurrent caches in Go. One version using `sync.Map` and another uses `singleflight`.

Usage:
- `go mod vendor`
- `go generate ./...`
- `go test -race`
