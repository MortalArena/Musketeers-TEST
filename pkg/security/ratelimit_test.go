package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 2.0,
		BurstSize:         5,
		CleanupInterval:   1 * time.Second,
		ExpiryDuration:    10 * time.Second,
	}

	rl := NewRateLimiter(config)

	ip := "192.168.1.1"

	// أول 5 طلبات يجب أن تنجح (burst)
	for i := 0; i < 5; i++ {
		allowed, _ := rl.Allow(ip)
		if !allowed {
			t.Errorf("الطلب %d يجب أن يُسمح به", i)
		}
	}

	// الطلب السادس يجب أن يرفض
	allowed, _ := rl.Allow(ip)
	if allowed {
		t.Error("الطلب السادس يجب أن يُرفض")
	}

	// انتظار ثانية واحدة
	time.Sleep(1 * time.Second)

	// الآن يجب السماح بطلبين
	for i := 0; i < 2; i++ {
		allowed, _ := rl.Allow(ip)
		if !allowed {
			t.Errorf("الطلب %d بعد الانتظار يجب أن يُسمح به", i)
		}
	}
}

func TestBlacklist(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())

	ip := "10.0.0.1"
	rl.BlockIP(ip)

	allowed, _ := rl.Allow(ip)
	if allowed {
		t.Error("IP المحظور يجب أن يُرفض")
	}

	rl.UnblockIP(ip)
	allowed, _ = rl.Allow(ip)
	if !allowed {
		t.Error("IP بعد إلغاء الحظر يجب أن يُسمح به")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 1.0,
		BurstSize:         2,
	}

	rl := NewRateLimiter(config)
	handler := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// أول طلبين يجب أن ينجحا
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("الطلب %d يجب أن ينجح", i)
		}
	}

	// الطلب الثالث يجب أن يرفض
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("الطلب الثالث يجب أن يُرفض، الكود: %d", w.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"},
			expected: "1.2.3.4",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "10.0.0.1"},
			expected: "10.0.0.1",
		},
		{
			name:     "RemoteAddr",
			remote:   "192.168.1.1:1234",
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if tt.remote != "" {
				req.RemoteAddr = tt.remote
			}

			ip := GetClientIP(req)
			if ip != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	rl := NewRateLimiter(DefaultRateLimitConfig())
	ip := "192.168.1.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow(ip)
	}
}
