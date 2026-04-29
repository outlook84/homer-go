# Development

homer-go is a Go server with server-side-rendered HTML, embedded assets, and server-side smart-card collectors. It is not a Vue application, so Homer Vue component and SCSS development instructions do not apply here.

## Project Layout

| Path | Purpose |
| --- | --- |
| `main.go` | Server setup, routes, collector registration, embedded assets. |
| `internal/config` | YAML loading, config normalization, page merging, message resolution. |
| `internal/collectors` | Smart-card collectors and HTTP helpers. |
| `internal/views` | HTML rendering helpers. |
| `assets` | Embedded CSS, JS, icons, themes, vendor assets, and example config. |
| `docs` | User-facing documentation and screenshots. |

## Running Locally

```sh
go run . -addr :8732 -data ./tmp
```

Create the data directory first. If `./tmp/config.yml` does not exist, homer-go can initialize it from the embedded example config.

## Tests

```sh
go test ./...
```

## Adding a Smart Card

1. Add a collector in `internal/collectors`.
2. Implement:

```go
type MyService struct{}

func (MyService) Type() string { return "MyService" }

func (MyService) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
    // Collect data and return a Status.
}
```

3. Register it in `main.go`.
4. Add tests in `internal/collectors/collectors_test.go` or a focused test file.
5. Document the new type in [smart-cards.md](./smart-cards.md).

Use `collectJSON`, `requestOptions`, `stringField`, and `effectiveHeaders` for consistency with existing collectors.

## Configuration Compatibility

The parser keeps the raw YAML map on `config.Config.Raw`, group `Raw`, and item `Raw`. This allows collectors to read service-specific keys without adding every option to the shared `Item` struct.

Prefer preserving Homer-compatible key names when they are sensible, such as `apikey`, `endpoint`, `username`, and `password`.
