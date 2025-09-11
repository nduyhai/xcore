package envloader

import (
	"os"
	"sync"
	"testing"
)

type TestConfig struct {
	Server struct {
		Host string `env:"SERVER_HOST"`
		Port int    `env:"SERVER_PORT"`
	}
	Database struct {
		URL      string `env:"DATABASE_URL"`
		Username string `env:"DATABASE_USERNAME"`
		Password string `env:"DATABASE_PASSWORD"`
	}
	Debug bool `env:"DEBUG"`
}

func TestLoad_SuccessfulLoadFromEnv(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Set environment variables
	os.Setenv("SERVER_HOST", "localhost")
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("DATABASE_URL", "postgres://localhost:5432/testdb")
	os.Setenv("DATABASE_USERNAME", "testuser")
	os.Setenv("DATABASE_PASSWORD", "testpass")
	os.Setenv("DEBUG", "true")
	
	defer func() {
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("DATABASE_USERNAME")
		os.Unsetenv("DATABASE_PASSWORD")
		os.Unsetenv("DEBUG")
	}()

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
	if config.Database.Username != "testuser" {
		t.Errorf("Expected database.username=testuser, got: %s", config.Database.Username)
	}
	if config.Database.Password != "testpass" {
		t.Errorf("Expected database.password=testpass, got: %s", config.Database.Password)
	}
	if !config.Debug {
		t.Errorf("Expected debug=true, got: %t", config.Debug)
	}
}

func TestLoad_PartialEnvVars(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Set only some environment variables
	os.Setenv("SERVER_HOST", "production.example.com")
	os.Setenv("DEBUG", "false")
	
	defer func() {
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("DEBUG")
	}()

	var config TestConfig
	err := Load(&config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify loaded values (unset values should remain zero)
	if config.Server.Host != "production.example.com" {
		t.Errorf("Expected server.host=production.example.com, got: %s", config.Server.Host)
	}
	if config.Server.Port != 0 {
		t.Errorf("Expected server.port=0 (unset), got: %d", config.Server.Port)
	}
	if config.Debug {
		t.Errorf("Expected debug=false, got: %t", config.Debug)
	}
}

func TestLoad_NoEnvVars(t *testing.T) {
	// Reset global state
	resetGlobalState()

	var config TestConfig
	err := Load(&config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all values remain zero/empty
	if config.Server.Host != "" {
		t.Errorf("Expected server.host to be empty, got: %s", config.Server.Host)
	}
	if config.Server.Port != 0 {
		t.Errorf("Expected server.port=0, got: %d", config.Server.Port)
	}
	if config.Database.URL != "" {
		t.Errorf("Expected database.url to be empty, got: %s", config.Database.URL)
	}
	if config.Debug {
		t.Errorf("Expected debug=false (zero value), got: %t", config.Debug)
	}
}

func TestLoad_WeaklyTypedInput(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Use string values in environment variables that should be converted to appropriate types
	os.Setenv("SERVER_HOST", "localhost")
	os.Setenv("SERVER_PORT", "8080")  // String that should convert to int
	os.Setenv("DEBUG", "true")        // String that should convert to bool
	
	defer func() {
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DEBUG")
	}()

	var config TestConfig
	err := Load(&config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify type conversion worked
	if config.Server.Port != 8080 {
		t.Errorf("Expected server.port=8080 (converted from string), got: %d", config.Server.Port)
	}
	if !config.Debug {
		t.Errorf("Expected debug=true (converted from string), got: %t", config.Debug)
	}
}

func TestLoad_NilDestination(t *testing.T) {
	// Reset global state
	resetGlobalState()

	err := Load(nil)

	if err == nil {
		t.Fatal("Expected error for nil destination, got nil")
	}

	if !containsString(err.Error(), "envloader: Load called with nil destination") {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestLoad_InvalidTypeConversion(t *testing.T) {
	// Reset global state
	resetGlobalState()

	// Set invalid port value that can't be converted to int
	os.Setenv("SERVER_PORT", "not-a-number")
	
	defer func() {
		os.Unsetenv("SERVER_PORT")
	}()

	var config TestConfig
	err := Load(&config)

	if err == nil {
		t.Fatal("Expected error for invalid type conversion, got nil")
	}

	if !containsString(err.Error(), "envloader: parse:") {
		t.Errorf("Expected parse error, got: %s", err.Error())
	}
}

// Helper functions
func resetGlobalState() {
	once = sync.Once{}
	initErr = nil
}

func containsString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}