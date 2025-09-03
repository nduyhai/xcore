package viperloader

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

var (
	once      sync.Once
	initErr   error
	vSnapshot *viper.Viper // immutable config vSnapshot after first init
)

func Load(dst any) error {
	once.Do(func() {
		v := viper.New()

		// 1) Base YAML (ignore if missing)
		v.SetConfigFile("config/config.yaml")
		if err := v.MergeInConfig(); err != nil {
			// ignore if a file not found
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if !errors.As(err, &configFileNotFoundError) {
				initErr = fmt.Errorf("viper: read yaml: %w", err)
			}
		}

		// 2) .env overrides (ignore if missing)
		v.SetConfigFile(".env")
		v.SetConfigType("env")
		if err := v.MergeInConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if !errors.As(err, &configFileNotFoundError) {
				initErr = fmt.Errorf("viper: read .env: %w", err)
			}
		}
		// 3) Process env (highest precedence)
		v.AutomaticEnv()

		// Keep the vSnapshot for reuse
		vSnapshot = v
	})

	if initErr != nil {
		return initErr
	}
	// Decode from the cached vSnapshot into the provided struct
	if err := vSnapshot.Unmarshal(dst, func(c *mapstructure.DecoderConfig) {
		c.TagName = "yaml"        // or "mapstructure"
		c.WeaklyTypedInput = true // "8080" -> int
	}); err != nil {
		return fmt.Errorf("viperloader: unmarshal: %w", err)
	}
	return nil
}
