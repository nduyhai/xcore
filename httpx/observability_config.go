package httpx

type ObsMode string

const (
	ObsModeManual     ObsMode = "manual"     // your current initTracing + initMetrics logic
	ObsModeAutoExport ObsMode = "autoexport" // uses OTEL_* env vars
)

type ObservabilityConfig struct {
	Mode    ObsMode
	Enabled bool
}

func NewObservabilityConfig(mode ObsMode) *ObservabilityConfig {
	return &ObservabilityConfig{Mode: mode}
}
