package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host 0.0.0.0, got %s", config.Server.Host)
	}
	
	if config.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.Port)
	}
	
	if config.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Expected read timeout 30s, got %v", config.Server.ReadTimeout)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid port",
			config: &Config{
				Server: ServerConfig{
					Port: 70000,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid SMTP port",
			config: &Config{
				Email: EmailConfig{
					SMTPPort: 70000,
				},
			},
			wantErr: true,
		},
		{
			name: "empty SMTP host",
			config: &Config{
				Email: EmailConfig{
					SMTPHost: "",
				},
			},
			wantErr: true,
		},
		{
			name: "empty storage path",
			config: &Config{
				Storage: StorageConfig{
					Path: "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid storage size",
			config: &Config{
				Storage: StorageConfig{
					Path:    "./data",
					MaxSize: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "TLS enabled without cert",
			config: &Config{
				Security: SecurityConfig{
					EnableTLS:  true,
					TLSCertFile: "",
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	config := DefaultConfig()
	config.Server.Port = 9090
	
	err := SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if loaded.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", loaded.Server.Port)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	config := DefaultConfig()
	config.Server.Port = 9090
	
	err := SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Check file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	
	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}
	
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
