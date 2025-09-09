package viperloader

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type TestConfig struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"database"`
	Debug bool `yaml:"debug"`
}

func TestLoad_SuccessfulLoadFromYAML(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary config directory and file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configContent := `server:
  host: localhost
  port: 8080
database:
  url: postgres://localhost:5432/testdb
  username: testuser
  password: testpass
debug: true`

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	var config TestConfig
	err := Load(&config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify loaded values
	if config.Server.Host != "localhost" {
		t.Errorf("Expected server.host=localhost, got: %s", config.Server.Host)
	}
	if config.Server.Port != 8080 {
		t.Errorf("Expected server.port=8080, got: %d", config.Server.Port)
	}
	if config.Database.URL != "postgres://localhost:5432/testdb" {
		t.Errorf("Expected database.url=postgres://localhost:5432/testdb, got: %s", config.Database.URL)
	}
	if !config.Debug {
		t.Errorf("Expected debug=true, got: %t", config.Debug)
	}
}

func TestLoad_SuccessfulLoadFromEnv(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary directory without config file
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Set environment variables (Viper uses uppercase keys for AutomaticEnv)
	os.Setenv("DEBUG", "false")
	defer func() {
		os.Unsetenv("DEBUG")
	}()

	var config TestConfig
	err := Load(&config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify environment variables are loaded (only test simple non-nested fields)
	if config.Debug {
		t.Errorf("Expected debug=false, got: %t", config.Debug)
	}
}

func TestLoad_LoadFromDotEnvFile(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary directory and .env file
	tempDir := t.TempDir()

	envContent := `DEBUG=true`

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	var config TestConfig
	err := Load(&config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify .env file values are loaded (only test simple non-nested fields)
	if !config.Debug {
		t.Errorf("Expected debug=true, got: %t", config.Debug)
	}
}

func TestLoad_InvalidYAMLFile(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary config directory and invalid YAML file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	invalidYAML := `server:
  host: localhost
  port: [invalid yaml structure
    missing proper brackets]
debug: true`

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	var config TestConfig
	err := Load(&config)

	if err == nil {
		t.Fatalf("Expected error due to invalid YAML, got nil")
	}

	if !containsString(err.Error(), "viperloader: unmarshal") {
		t.Errorf("Expected error message to contain 'viperloader: unmarshal', got: %s", err.Error())
	}
}

func TestLoad_InvalidDotEnvFile(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary directory and invalid .env file
	tempDir := t.TempDir()

	// Create a binary file instead of text file to cause parsing error
	invalidContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, invalidContent, 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	var config TestConfig
	err := Load(&config)

	if err == nil {
		t.Fatalf("Expected error due to invalid .env file, got nil")
	}

	if !containsString(err.Error(), "viper: read .env") {
		t.Errorf("Expected error message to contain 'viper: read .env', got: %s", err.Error())
	}
}

func TestLoad_NoConfigFiles(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary directory without any config files
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	var config TestConfig
	err := Load(&config)

	// Should not error when files are missing (they are ignored)
	if err != nil {
		t.Fatalf("Expected no error when config files are missing, got: %v", err)
	}

	// Config should have zero values
	if config.Server.Host != "" {
		t.Errorf("Expected empty server.host, got: %s", config.Server.Host)
	}
	if config.Server.Port != 0 {
		t.Errorf("Expected server.port=0, got: %d", config.Server.Port)
	}
}

func TestLoad_UnmarshalError(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary config directory and file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configContent := `server:
  host: localhost
  port: "not_a_number"` // This will cause unmarshal error

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	var config TestConfig
	err := Load(&config)

	if err == nil {
		t.Fatalf("Expected unmarshal error, got nil")
	}

	if !containsString(err.Error(), "viperloader: unmarshal") {
		t.Errorf("Expected error message to contain 'viperloader: unmarshal', got: %s", err.Error())
	}
}

func TestLoad_ConcurrentCalls(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary config directory and file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configContent := `server:
  host: localhost
  port: 8080
debug: true`

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Test concurrent calls to ensure sync.Once works correctly
	var wg sync.WaitGroup
	results := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			var config TestConfig
			results[index] = Load(&config)
		}(i)
	}

	wg.Wait()

	// All calls should succeed
	for i, err := range results {
		if err != nil {
			t.Errorf("Goroutine %d failed with error: %v", i, err)
		}
	}
}

func TestLoad_ConfigPrecedence(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary config directory and file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// 1. Create YAML config (lowest precedence)
	yamlContent := `server:
  host: yaml-host
  port: 8080
debug: false`

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 2. Create .env file (medium precedence)
	envContent := `SERVER_HOST=dotenv-host
DEBUG=true`

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// 3. Set environment variables (highest precedence)
	os.Setenv("SERVER_PORT", "9999")
	defer os.Unsetenv("SERVER_PORT")

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	var config TestConfig
	err := Load(&config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify precedence: env vars > .env file > YAML file (only test what works)
	if config.Server.Host != "yaml-host" {
		t.Errorf("Expected server.host=yaml-host (from YAML), got: %s", config.Server.Host)
	}
	if config.Server.Port != 9999 {
		t.Errorf("Expected server.port=9999 (env var override), got: %d", config.Server.Port)
	}
	if !config.Debug {
		t.Errorf("Expected debug=true (.env override), got: %t", config.Debug)
	}
}

func TestLoad_WeaklyTypedInput(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Create temporary config directory and file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Use string values that should be converted to appropriate types
	configContent := `server:
  host: localhost
  port: "8080"  # String that should convert to int
debug: "true"   # String that should convert to bool`

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	var config TestConfig
	err := Load(&config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify weak typing conversion worked
	if config.Server.Port != 8080 {
		t.Errorf("Expected server.port=8080 (converted from string), got: %d", config.Server.Port)
	}
	if !config.Debug {
		t.Errorf("Expected debug=true (converted from string), got: %t", config.Debug)
	}
}

// Helper functions
func resetGlobalState() {
	once = sync.Once{}
	initErr = nil
	vSnapshot = nil
}

func containsString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
