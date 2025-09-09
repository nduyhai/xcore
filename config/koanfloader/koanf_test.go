package koanfloader

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
	
	// Set environment variables (Koanf env provider uses flat keys, not nested)
	os.Setenv("server.host", "envhost")
	os.Setenv("server.port", "9090")
	os.Setenv("debug", "false")
	defer func() {
		os.Unsetenv("server.host")
		os.Unsetenv("server.port")
		os.Unsetenv("debug")
	}()
	
	var config TestConfig
	err := Load(&config)
	
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Verify environment variables override
	if config.Server.Host != "envhost" {
		t.Errorf("Expected server.host=envhost, got: %s", config.Server.Host)
	}
	if config.Server.Port != 9090 {
		t.Errorf("Expected server.port=9090, got: %d", config.Server.Port)
	}
	if config.Debug {
		t.Errorf("Expected debug=false, got: %t", config.Debug)
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
  port: invalid yaml structure
    missing proper indentation
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
	
	if !containsString(err.Error(), "koanfloader: unmarshal") {
		t.Errorf("Expected error message to contain 'koanfloader: unmarshal', got: %s", err.Error())
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
  port: "not_a_number"`  // This will cause unmarshal error
	
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
	
	if !containsString(err.Error(), "koanfloader: unmarshal") {
		t.Errorf("Expected error message to contain 'koanfloader: unmarshal', got: %s", err.Error())
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

func TestLoad_YAMLAndEnvPrecedence(t *testing.T) {
	// Reset global state
	resetGlobalState()
	
	// Create temporary config directory and file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}
	
	configContent := `server:
  host: yaml-host
  port: 8080
debug: false`
	
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)
	
	// Set environment variables that should override YAML (using dot notation)
	os.Setenv("server.host", "env-host")
	os.Setenv("debug", "true")
	defer func() {
		os.Unsetenv("server.host")
		os.Unsetenv("debug")
	}()
	
	var config TestConfig
	err := Load(&config)
	
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Environment variables should override YAML values
	if config.Server.Host != "env-host" {
		t.Errorf("Expected server.host=env-host (env override), got: %s", config.Server.Host)
	}
	if config.Server.Port != 8080 {
		t.Errorf("Expected server.port=8080 (from YAML), got: %d", config.Server.Port)
	}
	if !config.Debug {
		t.Errorf("Expected debug=true (env override), got: %t", config.Debug)
	}
}

// Helper functions
func resetGlobalState() {
	once = sync.Once{}
	initErr = nil
	kSnapshot = nil
}

func containsString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}