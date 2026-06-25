package node

import (
	"os"
	"strconv"
)

// Config إعدادات العقدة
type Config struct {
	ListenPort      int
	EnableMDNS      bool
	DataDir         string
	MaxPutPerMinute int
	EnableTor       bool
	TorProxyAddr    string
	RESTPort        int
	StorageQuotaMB  int64
	BootstrapPeers  []string
	FounderPubHex   string // مفتاح المؤسس العام (hex)
}

// DefaultConfig إعدادات افتراضية
func DefaultConfig() *Config {
	return &Config{
		ListenPort:      4001,
		EnableMDNS:      true,
		DataDir:         "./data",
		MaxPutPerMinute: 300, // 5/sec — كافي لبدء التشغيل دون أخطاء معدل
		StorageQuotaMB:  1024,
		RESTPort:        8080,
	}
}

// LoadFromEnv يقرأ الإعدادات من متغيرات البيئة
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	if v := os.Getenv("NR_LISTEN_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.ListenPort = p
		}
	}
	if v := os.Getenv("NR_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("NR_MDNS"); v == "false" {
		cfg.EnableMDNS = false
	}
	if v := os.Getenv("NR_MAX_PUT_PER_MINUTE"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.MaxPutPerMinute = p
		}
	}
	if v := os.Getenv("NR_STORAGE_QUOTA_MB"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.StorageQuotaMB = p
		}
	}
	if v := os.Getenv("NR_REST_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.RESTPort = p
		}
	}
	if v := os.Getenv("NR_ENABLE_TOR"); v == "true" {
		cfg.EnableTor = true
	}
	if v := os.Getenv("NR_TOR_PROXY"); v != "" {
		cfg.TorProxyAddr = v
	}
	if v := os.Getenv("NR_FOUNDER_PUB"); v != "" {
		cfg.FounderPubHex = v
	}

	return cfg
}
