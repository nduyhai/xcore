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

const (
	errScope       = "koanfloader"
	keyDelimiter   = "."
	configFilePath = "config/config.yaml"
	envPrefix      = "." // koanf env provider will read dot-separated keys
)

var (
	once      sync.Once
	initErr   error
	kSnapshot *koanf.Koanf
)

// setInitErrIfPresent sets initErr when err is non-nil and not an os.ErrNotExist.
// Returns true when initErr was set (caller should return early).
func setInitErrIfPresent(err error, source string) bool {
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return false
	}
	initErr = fmt.Errorf("%s: read %s: %w", errScope, source, err)
	return true
}

func Load(dst any) error {
	if dst == nil {
		return fmt.Errorf("%s: Load called with nil destination", errScope)
	}

	once.Do(func() {
		k := koanf.New(keyDelimiter)

		// 1) Load YAML defaults (ignore if missing)
		if setInitErrIfPresent(k.Load(file.Provider(configFilePath), yaml.Parser()), "config.yaml") {
			return
		}

		// 2) Load .env style (KEY=VALUE pairs). Ignore if missing.
		if setInitErrIfPresent(k.Load(env.Provider(envPrefix, env.Opt{}), nil), ".env") {
			return
		}

		kSnapshot = k
	})

	if initErr != nil {
		return initErr
	}
	if err := kSnapshot.Unmarshal("", dst); err != nil {
		return fmt.Errorf("%s: unmarshal: %w", errScope, err)
	}
	return nil
}
