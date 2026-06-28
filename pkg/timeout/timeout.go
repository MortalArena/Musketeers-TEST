package timeout

import (
	"context"
	"fmt"
	"time"
)

// Default timeouts for various operations
const (
	DefaultDialTimeout      = 10 * time.Second
	DefaultReadTimeout      = 30 * time.Second
	DefaultWriteTimeout     = 30 * time.Second
	DefaultRequestTimeout   = 30 * time.Second
	DefaultConnectTimeout   = 10 * time.Second
	DefaultHandshakeTimeout = 15 * time.Second
	DefaultStreamTimeout    = 60 * time.Second
	DefaultDiscoveryTimeout = 30 * time.Second
)

// TimeoutConfig holds timeout configuration
type TimeoutConfig struct {
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	RequestTimeout   time.Duration
	ConnectTimeout   time.Duration
	HandshakeTimeout time.Duration
	StreamTimeout    time.Duration
	DiscoveryTimeout time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		DialTimeout:      DefaultDialTimeout,
		ReadTimeout:      DefaultReadTimeout,
		WriteTimeout:     DefaultWriteTimeout,
		RequestTimeout:   DefaultRequestTimeout,
		ConnectTimeout:   DefaultConnectTimeout,
		HandshakeTimeout: DefaultHandshakeTimeout,
		StreamTimeout:    DefaultStreamTimeout,
		DiscoveryTimeout: DefaultDiscoveryTimeout,
	}
}

// WithDialTimeout adds dial timeout to context
func WithDialTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultDialTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// WithReadTimeout adds read timeout to context
func WithReadTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultReadTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// WithWriteTimeout adds write timeout to context
func WithWriteTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultWriteTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// WithRequestTimeout adds request timeout to context
func WithRequestTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultRequestTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// WithConnectTimeout adds connect timeout to context
func WithConnectTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// WithHandshakeTimeout adds handshake timeout to context
func WithHandshakeTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultHandshakeTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// WithStreamTimeout adds stream timeout to context
func WithStreamTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultStreamTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// WithDiscoveryTimeout adds discovery timeout to context
func WithDiscoveryTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultDiscoveryTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// ExecuteWithTimeout executes a function with timeout
func ExecuteWithTimeout(ctx context.Context, timeout time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic: %v", r)
			}
		}()
		errCh <- fn()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("operation timed out after %v", timeout)
	}
}

// ExecuteWithTimeoutResult executes a function with timeout and returns result
func ExecuteWithTimeoutResult[T any](ctx context.Context, timeout time.Duration, fn func() (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		value T
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				var zero T
				resultCh <- result{value: zero, err: fmt.Errorf("panic: %v", r)}
			}
		}()
		value, err := fn()
		resultCh <- result{value: value, err: err}
	}()

	select {
	case res := <-resultCh:
		return res.value, res.err
	case <-ctx.Done():
		var zero T
		return zero, fmt.Errorf("operation timed out after %v", timeout)
	}
}

// ExecuteWithDeadline executes a function with deadline
func ExecuteWithDeadline(ctx context.Context, deadline time.Time, fn func() error) error {
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic: %v", r)
			}
		}()
		errCh <- fn()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("operation exceeded deadline")
	}
}

// RetryWithTimeout retries a function with timeout between attempts
func RetryWithTimeout(ctx context.Context, maxAttempts int, timeout time.Duration, fn func() error) error {
	var lastErr error

	timer := time.NewTimer(0)
	defer timer.Stop()
	if !timer.Stop() {
		<-timer.C
	}

	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			timer.Reset(timeout)
			select {
			case <-timer.C:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
	}
	
	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(ctx context.Context, maxAttempts int, initialBackoff time.Duration, fn func() error) error {
	var lastErr error
	backoff := initialBackoff

	timer := time.NewTimer(0)
	defer timer.Stop()
	if !timer.Stop() {
		<-timer.C
	}

	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			timer.Reset(backoff)
			select {
			case <-timer.C:
			case <-ctx.Done():
				return ctx.Err()
			}
			backoff *= 2 // Exponential backoff
		}
		
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
	}
	
	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}
