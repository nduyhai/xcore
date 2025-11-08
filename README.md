# xcore

[![Go](https://img.shields.io/badge/go-1.25+-blue)](https://go.dev/)
[![License](https://img.shields.io/github/license/nduyhai/xcore)](LICENSE)

`xcore` is a multi-module Go workspace that collects reusable building blocks for
service development. It includes structured error handling primitives,
configuration loaders, and Kafka client adapters that can be imported
individually in your projects.

## Packages

### Error handling

| Module | Summary |
| --- | --- |
| [`error/xerr`](error/xerr) | Foundational error interface with stack traces, metadata helpers, and HTTP-aware reasons. |
| [`error/gerr`](error/gerr) | Bridges `xerr` with gRPC by serialising reasons through protobuf and mapping `codes.Code` values. |
| [`error/xgen`](error/xgen) | YAML-driven generator that emits Go, gRPC, and HTTP error definitions for consistent code creation. |

### Configuration loaders

| Module | Summary |
| --- | --- |
| [`config/envloader`](config/envloader) | Lightweight wrapper around `caarlos0/env` with optional `.env` support. |
| [`config/koanfloader`](config/koanfloader) | Opinionated loader that merges `config/config.yaml`, `.env`, and process variables using Koanf. |
| [`config/viperloader`](config/viperloader) | Cached Viper instance with automatic environment overrides and `mapstructure` decoding. |

### Kafka utilities

| Module | Summary |
| --- | --- |
| [`pubsub/kafkit`](pubsub/kafkit) | Core interfaces, configuration, and registration helpers shared by Kafka implementations. |
| [`pubsub/franzgo`](pubsub/franzgo) | Adapter backed by [`franz-go`](https://github.com/twmb/franz-go) clients. |
| [`pubsub/segmentio`](pubsub/segmentio) | Adapter backed by [`segmentio/kafka-go`](https://github.com/segmentio/kafka-go). |

## Getting started

Each package is versioned independently. Add the ones you need via `go get`:

```bash
go get github.com/nduyhai/xcore/error/xerr@latest
```

Use the shared `Makefile` targets when working inside the workspace:

- `make modules` – list detected modules in the repo
- `make test` – run unit and integration tests for every module (use `MODULE=path` to scope)
- `make lint` – execute `golangci-lint` across modules
- `make xgen` – regenerate error definitions from [`error/xgen/errors.yaml`](error/xgen/errors.yaml)

### Example: structured errors

```go
package users

import (
        "fmt"

        "github.com/nduyhai/xcore/error/xerr"
)

type User struct {
        ID   string
        Name string
}

func findUser(id string) (User, error) {
        // ... lookup logic
        reason := xerr.NewHTTPReason("USER_NOT_FOUND", fmt.Sprintf("user %s not found", id), 404)
        return User{}, xerr.New(reason, nil).WithMetadata("user_id", id)
}
```

## Development

This repository uses Go workspaces (`go.work`) to coordinate modules. Running
`go test ./...` from the root will iterate through every module automatically via
the provided Makefile. Refer to `make help` for the full command list.

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for more information.
