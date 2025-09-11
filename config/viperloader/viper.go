package viperloader

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const (
	dotEnvPath = ".env"
	decoderTag = "mapstructure"
)

var (
	once      sync.Once
	initErr   error
	vSnapshot *viper.Viper // immutable config snapshot after first init
)

// Load initializes a cached Viper instance once (.env -> process env),
// then unmarshals the merged configuration into dst on every call.
func Load(dst any) error {
	once.Do(func() {
		v := viper.New()

		// Configure environment variable handling
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
		v.AllowEmptyEnv(true)
		v.AutomaticEnv()
		
		// 1) .env file (ignore if missing)
		if err := mergeConfigIgnoreNotFound(v, dotEnvPath, "env", "viper: read .env"); err != nil {
			initErr = err
			return
		}

		// Keep the snapshot for reuse
		vSnapshot = v
	})

	if initErr != nil {
		return initErr
	}

	// Decode from the cached snapshot into the provided struct
	if err := vSnapshot.Unmarshal(dst, func(c *mapstructure.DecoderConfig) {
		c.TagName = decoderTag    // or "mapstructure"
		c.WeaklyTypedInput = true // "8080" -> int
	}); err != nil {
		return fmt.Errorf("viperloader: unmarshal: %w", err)
	}

	return nil
}

// mergeConfigIgnoreNotFound sets the config file (and optional type), merges it,
// and returns nil if the file is missing, or a wrapped error otherwise.
func mergeConfigIgnoreNotFound(v *viper.Viper, path, cfgType, errPrefix string) error {
	v.SetConfigFile(path)
	if cfgType != "" {
		v.SetConfigType(cfgType)
	}
	if err := v.MergeInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%s: %w", errPrefix, err)
		}
	}
	return nil
}
