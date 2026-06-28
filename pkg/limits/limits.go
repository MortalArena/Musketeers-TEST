package limits

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceLimiter manages resource limits for operations
type ResourceLimiter struct {
	maxConcurrent int32
	current       int32
	semaphore     chan struct{}
	mu            sync.Mutex
}

// NewResourceLimiter creates a new resource limiter
func NewResourceLimiter(maxConcurrent int) *ResourceLimiter {
	return &ResourceLimiter{
		maxConcurrent: int32(maxConcurrent),
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// Acquire acquires a resource slot
func (rl *ResourceLimiter) Acquire(ctx context.Context) error {
	select {
	case rl.semaphore <- struct{}{}:
		atomic.AddInt32(&rl.current, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases a resource slot
func (rl *ResourceLimiter) Release() {
	<-rl.semaphore
	atomic.AddInt32(&rl.current, -1)
}

// Current returns current resource usage
func (rl *ResourceLimiter) Current() int32 {
	return atomic.LoadInt32(&rl.current)
}

// Max returns maximum concurrent resources
func (rl *ResourceLimiter) Max() int32 {
	return atomic.LoadInt32(&rl.maxConcurrent)
}

// SetMax sets maximum concurrent resources
func (rl *ResourceLimiter) SetMax(max int) {
	atomic.StoreInt32(&rl.maxConcurrent, int32(max))
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	newSemaphore := make(chan struct{}, max)
	for i := 0; i < len(rl.semaphore); i++ {
		newSemaphore <- struct{}{}
	}
	rl.semaphore = newSemaphore
}

// MemoryLimiter manages memory limits
type MemoryLimiter struct {
	maxMemory    int64
	currentUsage int64
	mu           sync.Mutex
}

// NewMemoryLimiter creates a new memory limiter
func NewMemoryLimiter(maxMemory int64) *MemoryLimiter {
	return &MemoryLimiter{
		maxMemory: maxMemory,
	}
}

// Allocate allocates memory
func (ml *MemoryLimiter) Allocate(size int64) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	if ml.currentUsage+size > ml.maxMemory {
		return fmt.Errorf("memory limit exceeded: %d/%d bytes", ml.currentUsage+size, ml.maxMemory)
	}
	
	ml.currentUsage += size
	return nil
}

// Release releases allocated memory
func (ml *MemoryLimiter) Release(size int64) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	ml.currentUsage -= size
	if ml.currentUsage < 0 {
		ml.currentUsage = 0
	}
}

// CurrentUsage returns current memory usage
func (ml *MemoryLimiter) CurrentUsage() int64 {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	return ml.currentUsage
}

// MaxMemory returns maximum memory limit
func (ml *MemoryLimiter) MaxMemory() int64 {
	return ml.maxMemory
}

// Available returns available memory
func (ml *MemoryLimiter) Available() int64 {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	return ml.maxMemory - ml.currentUsage
}

// RateLimiter manages rate limiting
type RateLimiter struct {
	tokens      int32
	maxTokens   int32
	refillRate  int32
	refillInterval time.Duration
	mu          sync.Mutex
	lastRefill  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int32, refillRate int32, refillInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:        maxTokens,
		maxTokens:     maxTokens,
		refillRate:    refillRate,
		refillInterval: refillInterval,
		lastRefill:    time.Now(),
	}
}

// Acquire acquires a token
func (rl *RateLimiter) Acquire(ctx context.Context) error {
	rl.refill()
	
	timer := time.NewTimer(rl.refillInterval)
	defer timer.Stop()
	
	for {
		rl.mu.Lock()
		if rl.tokens > 0 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}
		rl.mu.Unlock()
		
		select {
		case <-timer.C:
			rl.refill()
			timer.Reset(rl.refillInterval)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// TryAcquire tries to acquire a token without blocking
func (rl *RateLimiter) TryAcquire() bool {
	rl.refill()
	
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// refill refills tokens based on time elapsed
func (rl *RateLimiter) refill() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	
	if elapsed >= rl.refillInterval {
		tokensToAdd := int32(elapsed / rl.refillInterval) * rl.refillRate
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
}

// Tokens returns current token count
func (rl *RateLimiter) Tokens() int32 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tokens
}

// MaxTokens returns maximum token count
func (rl *RateLimiter) MaxTokens() int32 {
	return rl.maxTokens
}

// ConnectionLimiter manages connection limits
type ConnectionLimiter struct {
	maxConnections int32
	current        int32
	connections    map[string]time.Time
	mu             sync.Mutex
}

// NewConnectionLimiter creates a new connection limiter
func NewConnectionLimiter(maxConnections int) *ConnectionLimiter {
	return &ConnectionLimiter{
		maxConnections: int32(maxConnections),
		connections:    make(map[string]time.Time),
	}
}

// Acquire acquires a connection slot
func (cl *ConnectionLimiter) Acquire(id string) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	
	if atomic.LoadInt32(&cl.current) >= cl.maxConnections {
		return fmt.Errorf("connection limit reached: %d/%d", cl.current, cl.maxConnections)
	}
	
	cl.connections[id] = time.Now()
	atomic.AddInt32(&cl.current, 1)
	return nil
}

// Release releases a connection slot
func (cl *ConnectionLimiter) Release(id string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	
	if _, exists := cl.connections[id]; exists {
		delete(cl.connections, id)
		atomic.AddInt32(&cl.current, -1)
	}
}

// Current returns current connection count
func (cl *ConnectionLimiter) Current() int32 {
	return atomic.LoadInt32(&cl.current)
}

// Max returns maximum connection count
func (cl *ConnectionLimiter) Max() int32 {
	return atomic.LoadInt32(&cl.maxConnections)
}

// CleanupIdle removes idle connections older than duration
func (cl *ConnectionLimiter) CleanupIdle(maxIdle time.Duration) int {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	
	now := time.Now()
	count := 0
	
	for id, lastActive := range cl.connections {
		if now.Sub(lastActive) > maxIdle {
			delete(cl.connections, id)
			atomic.AddInt32(&cl.current, -1)
			count++
		}
	}
	
	return count
}

// UpdateActivity updates the last activity time for a connection
func (cl *ConnectionLimiter) UpdateActivity(id string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	
	if _, exists := cl.connections[id]; exists {
		cl.connections[id] = time.Now()
	}
}
