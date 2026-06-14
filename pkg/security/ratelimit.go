package security

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig إعدادات rate limiting
type RateLimitConfig struct {
	RequestsPerSecond float64       // RPS لكل IP
	BurstSize         int           // الحد الأقصى للانفجار
	CleanupInterval   time.Duration // فترة تنظيف الـ cache
	ExpiryDuration    time.Duration // مدة انتهاء صلاحية limiter
	Whitelist         []string      // IPs معفاة
	Blacklist         []string      // IPs محظورة
}

// DefaultRateLimitConfig الإعدادات الافتراضية
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerSecond: 10.0,         // 10 طلبات/ثانية
		BurstSize:         100,          // 100 طلب دفعة واحدة
		CleanupInterval:   5 * time.Minute,
		ExpiryDuration:    1 * time.Hour,
		Whitelist:         []string{},
		Blacklist:         []string{},
	}
}

// RateLimiter يدير حدود المعدل
type RateLimiter struct {
	limiters  map[string]*clientLimiter
	mu        sync.RWMutex
	config    *RateLimitConfig
	whitelist map[string]bool
	blacklist map[string]bool
}

// clientLimiter limiter لعميل واحد
type clientLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewRateLimiter ينشئ rate limiter جديد
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &RateLimiter{
		limiters:  make(map[string]*clientLimiter),
		config:    config,
		whitelist: make(map[string]bool),
		blacklist: make(map[string]bool),
	}

	// بناء whitelist
	for _, ip := range config.Whitelist {
		rl.whitelist[ip] = true
	}

	// بناء blacklist
	for _, ip := range config.Blacklist {
		rl.blacklist[ip] = true
	}

	// بدء cleanup دوري
	go rl.cleanup()

	return rl
}

// Allow يتحقق من السماح بالطلب
func (rl *RateLimiter) Allow(ip string) (bool, *rate.Limit) {
	// التحقق من blacklist
	if rl.blacklist[ip] {
		return false, nil
	}

	// التحقق من whitelist (معفى)
	if rl.whitelist[ip] {
		return true, nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists := rl.limiters[ip]
	if !exists {
		client = &clientLimiter{
			limiter: rate.NewLimiter(
				rate.Limit(rl.config.RequestsPerSecond),
				rl.config.BurstSize,
			),
		}
		rl.limiters[ip] = client
	}

	client.lastAccess = time.Now()
	return client.limiter.Allow(), nil
}

// cleanup ينظف الـ limiters القديمة
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, client := range rl.limiters {
			if now.Sub(client.lastAccess) > rl.config.ExpiryDuration {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// BlockIP يحظر IP
func (rl *RateLimiter) BlockIP(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.blacklist[ip] = true
	delete(rl.limiters, ip)
}

// UnblockIP يلغي حظر IP
func (rl *RateLimiter) UnblockIP(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.blacklist, ip)
}

// Stats يعرض إحصائيات
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"active_limiters": len(rl.limiters),
		"whitelist_size":  len(rl.whitelist),
		"blacklist_size":  len(rl.blacklist),
	}
}

// RateLimitMiddleware middleware للـ HTTP
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r)

			allowed, _ := limiter.Allow(ip)
			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.Header().Set("X-RateLimit-Limit", "10")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{
					"error": "تم تجاوز حد المعدل",
					"message": "يرجى المحاولة بعد 60 ثانية",
					"retry_after": 60
				}`))
				return
			}

			// إضافة rate limit headers
			w.Header().Set("X-RateLimit-Limit", "10")
			w.Header().Set("X-RateLimit-Remaining", "9")

			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP يستخرج IP العميل
func GetClientIP(r *http.Request) string {
	// 1. X-Forwarded-For (من proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 2. X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// 3. RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// EndpointRateLimiter rate limiting لكل endpoint
type EndpointRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	configs  map[string]*RateLimitConfig
}

// NewEndpointRateLimiter ينشئ endpoint limiter
func NewEndpointRateLimiter() *EndpointRateLimiter {
	erl := &EndpointRateLimiter{
		limiters: make(map[string]*RateLimiter),
		configs:  make(map[string]*RateLimitConfig),
	}

	// إعدادات خاصة لكل endpoint
	erl.configs["/api/auth/login"] = &RateLimitConfig{
		RequestsPerSecond: 1.0,  // أبطأ للتسجيل
		BurstSize:         5,
		CleanupInterval:   5 * time.Minute,
		ExpiryDuration:    1 * time.Hour,
	}

	erl.configs["/api/channels/messages"] = &RateLimitConfig{
		RequestsPerSecond: 20.0, // أسرع للرسائل
		BurstSize:         50,
		CleanupInterval:   5 * time.Minute,
		ExpiryDuration:    1 * time.Hour,
	}

	erl.configs["/api/identity/create"] = &RateLimitConfig{
		RequestsPerSecond: 0.1, // بطيء جداً لإنشاء هوية
		BurstSize:         1,
		CleanupInterval:   5 * time.Minute,
		ExpiryDuration:    1 * time.Hour,
	}

	return erl
}

// GetLimiterForEndpoint يعيد limiter للـ endpoint
func (erl *EndpointRateLimiter) GetLimiterForEndpoint(endpoint string) *RateLimiter {
	erl.mu.RLock()
	limiter, exists := erl.limiters[endpoint]
	erl.mu.RUnlock()

	if exists {
		return limiter
	}

	erl.mu.Lock()
	defer erl.mu.Unlock()

	// Double-check
	if limiter, exists := erl.limiters[endpoint]; exists {
		return limiter
	}

	// إنشاء limiter جديد
	config, ok := erl.configs[endpoint]
	if !ok {
		config = DefaultRateLimitConfig()
	}

	limiter = NewRateLimiter(config)
	erl.limiters[endpoint] = limiter
	return limiter
}
