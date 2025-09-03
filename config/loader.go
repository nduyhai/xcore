package config

import (
	"context"
)

type Loader interface {
	Load(ctx context.Context, path string, dst any) error
	Name() string
}
