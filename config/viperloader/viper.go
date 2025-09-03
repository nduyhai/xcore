package viperloader

import (
	"context"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Loader struct{}

func (Loader) Name() string { return "viper" }

func (Loader) Load(_ context.Context, _ string, dst any) error {
	v := viper.New()

	// 1. Load YAML (defaults)
	v.SetConfigFile("config/config.yaml")
	if err := v.MergeInConfig(); err != nil {
		// ignore if a file not found
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("viper: read yaml: %w", err)
		}
	}

	// 2. Load .env (KEY=VALUE pairs)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("viper: read .env: %w", err)
		}
	}

	// 🔹 3. Unmarshal into struct
	if err := v.Unmarshal(dst, func(c *mapstructure.DecoderConfig) {
		c.TagName = "yaml"        // unify with yaml tags
		c.WeaklyTypedInput = true // allow string→int etc.
	}); err != nil {
		return fmt.Errorf("viper: unmarshal: %w", err)
	}

	return nil
}
