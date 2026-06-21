package limits

import (
	"context"
	"testing"
	"time"
)

func TestResourceLimiter_AcquireRelease(t *testing.T) {
	limiter := NewResourceLimiter(2)
	
	ctx := context.Background()
	
	// Acquire first slot
	err := limiter.Acquire(ctx)
	if err != nil {
		t.Errorf("Failed to acquire first slot: %v", err)
	}
	
	if limiter.Current() != 1 {
		t.Errorf("Expected current 1, got %d", limiter.Current())
	}
	
	// Acquire second slot
	err = limiter.Acquire(ctx)
	if err != nil {
		t.Errorf("Failed to acquire second slot: %v", err)
	}
	
	if limiter.Current() != 2 {
		t.Errorf("Expected current 2, got %d", limiter.Current())
	}
	
	// Release first slot
	limiter.Release()
	
	if limiter.Current() != 1 {
		t.Errorf("Expected current 1, got %d", limiter.Current())
	}
	
	// Release second slot
	limiter.Release()
	
	if limiter.Current() != 0 {
		t.Errorf("Expected current 0, got %d", limiter.Current())
	}
}

func TestResourceLimiter_Max(t *testing.T) {
	limiter := NewResourceLimiter(5)
	
	if limiter.Max() != 5 {
		t.Errorf("Expected max 5, got %d", limiter.Max())
	}
}

func TestResourceLimiter_SetMax(t *testing.T) {
	limiter := NewResourceLimiter(2)
	limiter.SetMax(5)
	
	if limiter.Max() != 5 {
		t.Errorf("Expected max 5, got %d", limiter.Max())
	}
}

func TestMemoryLimiter_AllocateRelease(t *testing.T) {
	limiter := NewMemoryLimiter(1000)
	
	// Allocate 500 bytes
	err := limiter.Allocate(500)
	if err != nil {
		t.Errorf("Failed to allocate: %v", err)
	}
	
	if limiter.CurrentUsage() != 500 {
		t.Errorf("Expected current usage 500, got %d", limiter.CurrentUsage())
	}
	
	// Allocate another 300 bytes
	err = limiter.Allocate(300)
	if err != nil {
		t.Errorf("Failed to allocate: %v", err)
	}
	
	if limiter.CurrentUsage() != 800 {
		t.Errorf("Expected current usage 800, got %d", limiter.CurrentUsage())
	}
	
	// Release 300 bytes
	limiter.Release(300)
	
	if limiter.CurrentUsage() != 500 {
		t.Errorf("Expected current usage 500, got %d", limiter.CurrentUsage())
	}
	
	// Release remaining 500 bytes
	limiter.Release(500)
	
	if limiter.CurrentUsage() != 0 {
		t.Errorf("Expected current usage 0, got %d", limiter.CurrentUsage())
	}
}

func TestMemoryLimiter_ExceedsLimit(t *testing.T) {
	limiter := NewMemoryLimiter(1000)
	
	err := limiter.Allocate(1500)
	if err == nil {
		t.Error("Expected error when exceeding limit, got nil")
	}
}

func TestMemoryLimiter_Available(t *testing.T) {
	limiter := NewMemoryLimiter(1000)
	
	limiter.Allocate(300)
	
	if limiter.Available() != 700 {
		t.Errorf("Expected available 700, got %d", limiter.Available())
	}
}

func TestRateLimiter_Acquire(t *testing.T) {
	limiter := NewRateLimiter(10, 5, 1*time.Second)
	
	ctx := context.Background()
	
	// Acquire 5 tokens
	for i := 0; i < 5; i++ {
		err := limiter.Acquire(ctx)
		if err != nil {
			t.Errorf("Failed to acquire token %d: %v", i, err)
		}
	}
	
	if limiter.Tokens() != 5 {
		t.Errorf("Expected 5 tokens remaining, got %d", limiter.Tokens())
	}
}

func TestRateLimiter_TryAcquire(t *testing.T) {
	limiter := NewRateLimiter(10, 5, 1*time.Second)
	
	// Try acquire 5 tokens
	for i := 0; i < 5; i++ {
		if !limiter.TryAcquire() {
			t.Errorf("Failed to try acquire token %d", i)
		}
	}
	
	if limiter.Tokens() != 5 {
		t.Errorf("Expected 5 tokens remaining, got %d", limiter.Tokens())
	}
	
	// Try acquire when no tokens available
	if limiter.TryAcquire() {
		t.Error("Expected TryAcquire to fail when no tokens available")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	limiter := NewRateLimiter(5, 5, 100*time.Millisecond)
	
	ctx := context.Background()
	
	// Acquire all tokens
	for i := 0; i < 5; i++ {
		limiter.Acquire(ctx)
	}
	
	if limiter.Tokens() != 0 {
		t.Errorf("Expected 0 tokens, got %d", limiter.Tokens())
	}
	
	// Wait for refill
	time.Sleep(150 * time.Millisecond)
	
	limiter.refill()
	
	if limiter.Tokens() != 5 {
		t.Errorf("Expected 5 tokens after refill, got %d", limiter.Tokens())
	}
}

func TestConnectionLimiter_AcquireRelease(t *testing.T) {
	limiter := NewConnectionLimiter(2)
	
	// Acquire first connection
	err := limiter.Acquire("conn1")
	if err != nil {
		t.Errorf("Failed to acquire first connection: %v", err)
	}
	
	if limiter.Current() != 1 {
		t.Errorf("Expected current 1, got %d", limiter.Current())
	}
	
	// Acquire second connection
	err = limiter.Acquire("conn2")
	if err != nil {
		t.Errorf("Failed to acquire second connection: %v", err)
	}
	
	if limiter.Current() != 2 {
		t.Errorf("Expected current 2, got %d", limiter.Current())
	}
	
	// Try to acquire third connection (should fail)
	err = limiter.Acquire("conn3")
	if err == nil {
		t.Error("Expected error when exceeding connection limit, got nil")
	}
	
	// Release first connection
	limiter.Release("conn1")
	
	if limiter.Current() != 1 {
		t.Errorf("Expected current 1, got %d", limiter.Current())
	}
	
	// Release second connection
	limiter.Release("conn2")
	
	if limiter.Current() != 0 {
		t.Errorf("Expected current 0, got %d", limiter.Current())
	}
}

func TestConnectionLimiter_CleanupIdle(t *testing.T) {
	limiter := NewConnectionLimiter(10)
	
	// Add some connections
	limiter.Acquire("conn1")
	limiter.Acquire("conn2")
	
	// Update activity for conn1
	limiter.UpdateActivity("conn1")
	
	// Wait a bit
	time.Sleep(50 * time.Millisecond)
	
	// Cleanup idle connections (older than 10ms)
	count := limiter.CleanupIdle(10 * time.Millisecond)
	
	if count != 1 {
		t.Errorf("Expected to cleanup 1 connection, got %d", count)
	}
	
	if limiter.Current() != 1 {
		t.Errorf("Expected current 1, got %d", limiter.Current())
	}
}

func TestConnectionLimiter_UpdateActivity(t *testing.T) {
	limiter := NewConnectionLimiter(10)
	
	limiter.Acquire("conn1")
	
	// Update activity should not fail
	limiter.UpdateActivity("conn1")
	
	// Update activity for non-existent connection should not fail
	limiter.UpdateActivity("conn2")
}

func TestConnectionLimiter_Max(t *testing.T) {
	limiter := NewConnectionLimiter(5)
	
	if limiter.Max() != 5 {
		t.Errorf("Expected max 5, got %d", limiter.Max())
	}
}
