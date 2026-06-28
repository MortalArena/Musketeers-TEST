package hosting

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ============================================================
// Hosting Package - حزمة الاستضافة
// ============================================================

// HostingConfig تكوين الاستضافة
type HostingConfig struct {
	Domain         string
	HTTPPort       int
	HTTPSPort      int
	SSLCertPath    string
	SSLKeyPath     string
	EnableHTTPS    bool
	EnableHTTP     bool
	MaxConnections int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

// HostingServer خادم الاستضافة
type HostingServer struct {
	config     *HostingConfig
	httpServer *http.Server
	httpsServer *http.Server
	mu        sync.RWMutex
	running   bool
}

// NewHostingServer إنشاء خادم استضافة جديد
func NewHostingServer(config *HostingConfig) *HostingServer {
	return &HostingServer{
		config: config,
	}
}

// Start بدء تشغيل خادم الاستضافة
func (hs *HostingServer) Start() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.running {
		return fmt.Errorf("server is already running")
	}

	var wg sync.WaitGroup
	errors := make(chan error, 2)

	// بدء خادم HTTP إذا كان مفعلاً
	if hs.config.EnableHTTP {
		wg.Add(1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					_ = r
				}
			}()
			defer wg.Done()
			if err := hs.startHTTP(); err != nil {
				errors <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()
	}

	// بدء خادم HTTPS إذا كان مفعلاً
	if hs.config.EnableHTTPS {
		wg.Add(1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					_ = r
				}
			}()
			defer wg.Done()
			if err := hs.startHTTPS(); err != nil {
				errors <- fmt.Errorf("HTTPS server error: %w", err)
			}
		}()
	}

	// انتظار بدء جميع الخوادم
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		wg.Wait()
		close(errors)
	}()

	hs.running = true

	// التحقق من الأخطاء
	for err := range errors {
		if err != nil {
			hs.running = false
			return err
		}
	}

	return nil
}

// Stop إيقاف خادم الاستضافة
func (hs *HostingServer) Stop() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if !hs.running {
		return fmt.Errorf("server is not running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var errors []error

	// إيقاف خادم HTTP
	if hs.httpServer != nil {
		if err := hs.httpServer.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("HTTP server shutdown error: %w", err))
		}
	}

	// إيقاف خادم HTTPS
	if hs.httpsServer != nil {
		if err := hs.httpsServer.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("HTTPS server shutdown error: %w", err))
		}
	}

	hs.running = false

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}

	return nil
}

// startHTTP بدء خادم HTTP
func (hs *HostingServer) startHTTP() error {
	hs.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", hs.config.HTTPPort),
		ReadTimeout:  hs.config.ReadTimeout,
		WriteTimeout: hs.config.WriteTimeout,
		IdleTimeout:  hs.config.IdleTimeout,
	}

	if err := hs.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}

// startHTTPS بدء خادم HTTPS
func (hs *HostingServer) startHTTPS() error {
	hs.httpsServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", hs.config.HTTPSPort),
		ReadTimeout:  hs.config.ReadTimeout,
		WriteTimeout: hs.config.WriteTimeout,
		IdleTimeout:  hs.config.IdleTimeout,
	}

	if err := hs.httpsServer.ListenAndServeTLS(hs.config.SSLCertPath, hs.config.SSLKeyPath); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTPS server failed: %w", err)
	}

	return nil
}

// IsRunning التحقق مما إذا كان الخادم يعمل
func (hs *HostingServer) IsRunning() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.running
}

// GetConfig الحصول على تكوين الخادم
func (hs *HostingServer) GetConfig() *HostingConfig {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.config
}

// SetHandler تعيين معالج HTTP
func (hs *HostingServer) SetHandler(handler http.Handler) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.httpServer != nil {
		hs.httpServer.Handler = handler
	}
	if hs.httpsServer != nil {
		hs.httpsServer.Handler = handler
	}
}

// ============================================================
// Hosting Manager - مدير الاستضافة
// ============================================================

// HostingManager يدير خوادم الاستضافة المتعددة
type HostingManager struct {
	servers map[string]*HostingServer
	mu      sync.RWMutex
}

// NewHostingManager إنشاء مدير استضافة جديد
func NewHostingManager() *HostingManager {
	return &HostingManager{
		servers: make(map[string]*HostingServer),
	}
}

// AddServer إضافة خادم استضافة
func (hm *HostingManager) AddServer(name string, config *HostingConfig) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.servers[name]; exists {
		return fmt.Errorf("server with name '%s' already exists", name)
	}

	server := NewHostingServer(config)
	hm.servers[name] = server

	return nil
}

// RemoveServer إزالة خادم استضافة
func (hm *HostingManager) RemoveServer(name string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	server, exists := hm.servers[name]
	if !exists {
		return fmt.Errorf("server with name '%s' not found", name)
	}

	if server.IsRunning() {
		if err := server.Stop(); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
	}

	delete(hm.servers, name)
	return nil
}

// StartServer بدء تشغيل خادم استضافة
func (hm *HostingManager) StartServer(name string) error {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	server, exists := hm.servers[name]
	if !exists {
		return fmt.Errorf("server with name '%s' not found", name)
	}

	return server.Start()
}

// StopServer إيقاف خادم استضافة
func (hm *HostingManager) StopServer(name string) error {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	server, exists := hm.servers[name]
	if !exists {
		return fmt.Errorf("server with name '%s' not found", name)
	}

	return server.Stop()
}

// GetServer الحصول على خادم استضافة
func (hm *HostingManager) GetServer(name string) (*HostingServer, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	server, exists := hm.servers[name]
	if !exists {
		return nil, fmt.Errorf("server with name '%s' not found", name)
	}

	return server, nil
}

// ListServers قائمة جميع الخوادم
func (hm *HostingManager) ListServers() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var names []string
	for name := range hm.servers {
		names = append(names, name)
	}

	return names
}

// StartAll بدء تشغيل جميع الخوادم
func (hm *HostingManager) StartAll() error {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var errors []error

	for name, server := range hm.servers {
		if err := server.Start(); err != nil {
			errors = append(errors, fmt.Errorf("server '%s': %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}

	return nil
}

// StopAll إيقاف جميع الخوادم
func (hm *HostingManager) StopAll() error {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var errors []error

	for name, server := range hm.servers {
		if err := server.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("server '%s': %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}

	return nil
}
