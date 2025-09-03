package koanfloader

import (
	"context"
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Loader struct{}

func (Loader) Name() string { return "koanf" }

func (Loader) Load(_ context.Context, _ string, dst any) error {
	k := koanf.New(".")

	// 1. Load YAML (defaults)
	if err := k.Load(file.Provider("config/config.yaml"), yaml.Parser()); err != nil {
		// ignore if missing
	}

	// 2. Load .env (KEY=VALUE pairs)
	if err := k.Load(env.Provider(".", env.Opt{}), nil); err != nil {
		// ignore if missing
	}

	// 3. Unmarshal into struct
	if err := k.Unmarshal("", dst); err != nil {
		return fmt.Errorf("koanf: unmarshal: %w", err)
	}
	return nil
}
