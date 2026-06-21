package timeout

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()
	
	if config.DialTimeout != DefaultDialTimeout {
		t.Errorf("Expected DialTimeout %v, got %v", DefaultDialTimeout, config.DialTimeout)
	}
	
	if config.ReadTimeout != DefaultReadTimeout {
		t.Errorf("Expected ReadTimeout %v, got %v", DefaultReadTimeout, config.ReadTimeout)
	}
	
	if config.RequestTimeout != DefaultRequestTimeout {
		t.Errorf("Expected RequestTimeout %v, got %v", DefaultRequestTimeout, config.RequestTimeout)
	}
}

func TestWithDialTimeout(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := WithDialTimeout(ctx, 5*time.Second)
	defer cancel()
	
	select {
	case <-ctx.Done():
	case <-time.After(6 * time.Second):
		t.Error("Context should have timed out")
	}
}

func TestWithReadTimeout(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := WithReadTimeout(ctx, 1*time.Second)
	defer cancel()
	
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Error("Context should have timed out")
	}
}

func TestWithRequestTimeout(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := WithRequestTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	
	select {
	case <-ctx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Error("Context should have timed out")
	}
}

func TestExecuteWithTimeout_Success(t *testing.T) {
	ctx := context.Background()
	err := ExecuteWithTimeout(ctx, 1*time.Second, func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteWithTimeout_Timeout(t *testing.T) {
	ctx := context.Background()
	err := ExecuteWithTimeout(ctx, 100*time.Millisecond, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestExecuteWithTimeout_Error(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	err := ExecuteWithTimeout(ctx, 1*time.Second, func() error {
		return expectedErr
	})
	
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestExecuteWithTimeoutResult_Success(t *testing.T) {
	ctx := context.Background()
	result, err := ExecuteWithTimeoutResult[string](ctx, 1*time.Second, func() (string, error) {
		return "test", nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if result != "test" {
		t.Errorf("Expected result 'test', got %v", result)
	}
}

func TestExecuteWithTimeoutResult_Timeout(t *testing.T) {
	ctx := context.Background()
	_, err := ExecuteWithTimeoutResult[string](ctx, 100*time.Millisecond, func() (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "test", nil
	})
	
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestExecuteWithDeadline_Success(t *testing.T) {
	ctx := context.Background()
	deadline := time.Now().Add(1 * time.Second)
	err := ExecuteWithDeadline(ctx, deadline, func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteWithDeadline_Exceeded(t *testing.T) {
	ctx := context.Background()
	deadline := time.Now().Add(100 * time.Millisecond)
	err := ExecuteWithDeadline(ctx, deadline, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	
	if err == nil {
		t.Error("Expected deadline error, got nil")
	}
}

func TestRetryWithTimeout_Success(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	err := RetryWithTimeout(ctx, 3, 100*time.Millisecond, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("not yet")
		}
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithTimeout_Failure(t *testing.T) {
	ctx := context.Background()
	err := RetryWithTimeout(ctx, 3, 100*time.Millisecond, func() error {
		return errors.New("always fails")
	})
	
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	err := RetryWithBackoff(ctx, 3, 50*time.Millisecond, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("not yet")
		}
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_Failure(t *testing.T) {
	ctx := context.Background()
	err := RetryWithBackoff(ctx, 3, 50*time.Millisecond, func() error {
		return errors.New("always fails")
	})
	
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestRetryWithBackoff_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	err := RetryWithBackoff(ctx, 3, 50*time.Millisecond, func() error {
		return errors.New("test")
	})
	
	if err == nil {
		t.Error("Expected context error, got nil")
	}
}
