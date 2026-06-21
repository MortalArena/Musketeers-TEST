package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for the Musketeers system
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Email    EmailConfig    `yaml:"email"`
	Storage  StorageConfig  `yaml:"storage"`
	Network  NetworkConfig  `yaml:"network"`
	Security SecurityConfig `yaml:"security"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
	MaxConns     int           `yaml:"max_connections"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// EmailConfig represents email configuration
type EmailConfig struct {
	SMTPHost     string `yaml:"smtp_host"`
	SMTPPort     int    `yaml:"smtp_port"`
	SMTPUsername string `yaml:"smtp_username"`
	SMTPPassword string `yaml:"smtp_password"`
	UseTLS      bool   `yaml:"use_tls"`
	FromAddress  string `yaml:"from_address"`
	FromName     string `yaml:"from_name"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type       string `yaml:"type"`
	Path       string `yaml:"path"`
	MaxSize    int64  `yaml:"max_size"`
	QuotaLimit int64  `yaml:"quota_limit"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	ListenAddr    string        `yaml:"listen_addr"`
	BootstrapPeers []string      `yaml:"bootstrap_peers"`
	DialTimeout   time.Duration `yaml:"dial_timeout"`
	MaxPeers      int           `yaml:"max_peers"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	EncryptionKey string `yaml:"encryption_key"`
	EnableTLS     bool   `yaml:"enable_tls"`
	TLSCertFile   string `yaml:"tls_cert_file"`
	TLSKeyFile    string `yaml:"tls_key_file"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
			MaxConns:     100,
		},
		Database: DatabaseConfig{
			Type:     "badger",
			Host:     "localhost",
			Port:     27017,
			Database: "musketeers",
			Username: "",
			Password: "",
		},
		Email: EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			SMTPUsername: "",
			SMTPPassword: "",
			UseTLS:      true,
			FromAddress:  "noreply@musketeers.com",
			FromName:     "Musketeers",
		},
		Storage: StorageConfig{
			Type:       "badger",
			Path:       "./data/storage",
			MaxSize:    10 * 1024 * 1024 * 1024, // 10GB
			QuotaLimit: 1 * 1024 * 1024 * 1024,  // 1GB
		},
		Network: NetworkConfig{
			ListenAddr:    "/ip4/0.0.0.0/tcp/4001",
			BootstrapPeers: []string{},
			DialTimeout:   10 * time.Second,
			MaxPeers:      50,
		},
		Security: SecurityConfig{
			EncryptionKey: "",
			EnableTLS:     false,
			TLSCertFile:   "",
			TLSKeyFile:    "",
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	// Validate server config
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	if config.Server.ReadTimeout < 0 {
		return fmt.Errorf("invalid read timeout: %v", config.Server.ReadTimeout)
	}
	if config.Server.WriteTimeout < 0 {
		return fmt.Errorf("invalid write timeout: %v", config.Server.WriteTimeout)
	}
	if config.Server.MaxConns < 1 {
		return fmt.Errorf("invalid max connections: %d", config.Server.MaxConns)
	}

	// Validate database config
	if config.Database.Type == "" {
		return fmt.Errorf("database type cannot be empty")
	}

	// Validate email config
	if config.Email.SMTPHost == "" {
		return fmt.Errorf("SMTP host cannot be empty")
	}
	if config.Email.SMTPPort < 1 || config.Email.SMTPPort > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", config.Email.SMTPPort)
	}
	if config.Email.FromAddress == "" {
		return fmt.Errorf("from address cannot be empty")
	}

	// Validate storage config
	if config.Storage.Type == "" {
		return fmt.Errorf("storage type cannot be empty")
	}
	if config.Storage.Path == "" {
		return fmt.Errorf("storage path cannot be empty")
	}
	if config.Storage.MaxSize < 0 {
		return fmt.Errorf("invalid max size: %d", config.Storage.MaxSize)
	}
	if config.Storage.QuotaLimit < 0 {
		return fmt.Errorf("invalid quota limit: %d", config.Storage.QuotaLimit)
	}

	// Validate network config
	if config.Network.ListenAddr == "" {
		return fmt.Errorf("listen address cannot be empty")
	}
	if config.Network.DialTimeout < 0 {
		return fmt.Errorf("invalid dial timeout: %v", config.Network.DialTimeout)
	}
	if config.Network.MaxPeers < 1 {
		return fmt.Errorf("invalid max peers: %d", config.Network.MaxPeers)
	}

	// Validate security config
	if config.Security.EnableTLS {
		if config.Security.TLSCertFile == "" {
			return fmt.Errorf("TLS cert file cannot be empty when TLS is enabled")
		}
		if config.Security.TLSKeyFile == "" {
			return fmt.Errorf("TLS key file cannot be empty when TLS is enabled")
		}
	}

	return nil
}
