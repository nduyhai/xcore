package envloader

import (
	"fmt"
	"sync"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const (
	errScope = "envloader"
)

var (
	once    sync.Once
	initErr error
)

// Load parses environment variables into the provided struct using caarlos0/env.
// This follows the same pattern as viperloader and koanfloader but uses
// the env library which directly parses environment variables into structs
// using struct tags.
func Load(dst any) error {
	if dst == nil {
		return fmt.Errorf("%s: Load called with nil destination", errScope)
	}

	once.Do(func() {
		_ = godotenv.Load()
	})

	if initErr != nil {
		return initErr
	}

	// Parse environment variables directly into the destination struct
	if err := env.Parse(dst); err != nil {
		return fmt.Errorf("%s: parse: %w", errScope, err)
	}

	return nil
}
