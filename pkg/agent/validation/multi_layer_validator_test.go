package validation

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestMultiLayerValidator_NewMultiLayerValidator(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mlv := NewMultiLayerValidator(logger)

	if mlv == nil {
		t.Fatal("NewMultiLayerValidator returned nil")
	}

	if mlv.inputValidator == nil {
		t.Error("inputValidator is nil")
	}

	if mlv.executionValidator == nil {
		t.Error("executionValidator is nil")
	}

	if mlv.outputValidator == nil {
		t.Error("outputValidator is nil")
	}

	if mlv.recoveryManager == nil {
		t.Error("recoveryManager is nil")
	}
}

func TestMultiLayerValidator_ValidateInput(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mlv := NewMultiLayerValidator(logger)

	ctx := context.Background()
	input := "test input"

	// التحقق من المدخلات
	result, err := mlv.ValidateInput(ctx, input)
	if err != nil {
		t.Fatalf("ValidateInput failed: %v", err)
	}

	if !result.Valid {
		t.Error("Expected valid input, got invalid")
	}
}

func TestMultiLayerValidator_ValidateOutput(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mlv := NewMultiLayerValidator(logger)

	ctx := context.Background()
	output := "test output"

	// التحقق من المخرجات
	result, err := mlv.ValidateOutput(ctx, output)
	if err != nil {
		t.Fatalf("ValidateOutput failed: %v", err)
	}

	if !result.Valid {
		t.Error("Expected valid output, got invalid")
	}
}

func TestMultiLayerValidator_ValidateAll(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mlv := NewMultiLayerValidator(logger)

	ctx := context.Background()
	input := "test input"
	output := "test output"

	// التحقق من جميع الطبقات
	result, err := mlv.ValidateAll(ctx, input, nil, output)
	if err != nil {
		t.Fatalf("ValidateAll failed: %v", err)
	}

	if !result.Valid {
		t.Error("Expected valid result, got invalid")
	}
}

func TestMultiLayerValidator_RecoverFromFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mlv := NewMultiLayerValidator(logger)

	ctx := context.Background()
	failure := &Failure{
		ID:      "test-failure",
		Error:   nil,
		Context: make(map[string]interface{}),
	}

	// الاسترداد من الفشل
	result, err := mlv.RecoverFromFailure(ctx, failure)
	if err != nil {
		t.Fatalf("RecoverFromFailure failed: %v", err)
	}

	if result == nil {
		t.Error("Expected recovery result, got nil")
	}
}

func TestMultiLayerValidator_GetValidationSummary(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mlv := NewMultiLayerValidator(logger)

	// الحصول على ملخص التحقق
	summary := mlv.GetValidationSummary()
	if summary == nil {
		t.Error("Expected summary, got nil")
	}

	if summary["input_validator_enabled"] != true {
		t.Error("Expected input_validator_enabled to be true")
	}

	if summary["execution_validator_enabled"] != true {
		t.Error("Expected execution_validator_enabled to be true")
	}

	if summary["output_validator_enabled"] != true {
		t.Error("Expected output_validator_enabled to be true")
	}

	if summary["recovery_manager_enabled"] != true {
		t.Error("Expected recovery_manager_enabled to be true")
	}
}

// اختبارات الأمان
func TestMultiLayerValidator_Security_ConcurrentValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mlv := NewMultiLayerValidator(logger)

	ctx := context.Background()

	// اختبار التحقق المتزامن
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { recover() }()
			mlv.ValidateInput(ctx, "test input")
			done <- true
		}()
	}

	// انتظار جميع goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
