package rate

import (
	"sync"
	"time"
)

type Limiter struct {
	mu       sync.Mutex
	rate     int
	burst    int
	tokens   int
	lastTick time.Time
}

func NewLimiter(rate, burst int) *Limiter {
	return &Limiter{
		rate:     rate,
		burst:    burst,
		tokens:   burst,
		lastTick: time.Now(),
	}
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastTick)
	l.lastTick = now

	l.tokens += int(elapsed.Seconds() * float64(l.rate))
	if l.tokens > l.burst {
		l.tokens = l.burst
	}

	if l.tokens > 0 {
		l.tokens--
		return true
	}
	return false
}

func (l *Limiter) SetRate(rate int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rate = rate
}

func (l *Limiter) SetBurst(burst int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.burst = burst
}
