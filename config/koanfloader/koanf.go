package koanfloader

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var (
	once      sync.Once
	initErr   error
	kSnapshot *koanf.Koanf
)

func Load(dst any) error {
	once.Do(func() {
		k := koanf.New(".")
		// 1. Load YAML (defaults)
		if err := k.Load(file.Provider("config/config.yaml"), yaml.Parser()); err != nil {
			// ignore if missing
			if !errors.Is(err, os.ErrNotExist) {
				initErr = fmt.Errorf("koanfloader: read config.yaml: %w", err)
			}
		}

		// 2. Load .env (KEY=VALUE pairs)
		if err := k.Load(env.Provider(".", env.Opt{}), nil); err != nil {
			// ignore if missing
			if !errors.Is(err, os.ErrNotExist) {
				initErr = fmt.Errorf("koanfloader: read .env: %w", err)
			}
		}

		kSnapshot = k
	})

	if initErr != nil {
		return initErr
	}

	if err := kSnapshot.Unmarshal("", dst); err != nil {
		return fmt.Errorf("koanfloader: unmarshal: %w", err)
	}
	return nil
}
