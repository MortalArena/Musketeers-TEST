package logger

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	if logger == nil {
		t.Fatal("Logger is nil")
	}
	
	if logger.sugar == nil {
		t.Fatal("Sugar logger is nil")
	}
}

func TestNewProductionLogger(t *testing.T) {
	logger, err := NewProductionLogger()
	if err != nil {
		t.Fatalf("Failed to create production logger: %v", err)
	}
	
	if logger == nil {
		t.Fatal("Logger is nil")
	}
}

func TestNewDevelopmentLogger(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	if err != nil {
		t.Fatalf("Failed to create development logger: %v", err)
	}
	
	if logger == nil {
		t.Fatal("Logger is nil")
	}
}

func TestNewNopLogger(t *testing.T) {
	logger := NewNopLogger()
	
	if logger == nil {
		t.Fatal("Logger is nil")
	}
	
	// Should not panic
	logger.Info("test")
	logger.Error("test")
}

func TestLogger_WithField(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	newLogger := logger.WithField("key", "value")
	if newLogger == nil {
		t.Fatal("WithField returned nil")
	}
}

func TestLogger_WithFields(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	
	newLogger := logger.WithFields(fields)
	if newLogger == nil {
		t.Fatal("WithFields returned nil")
	}
}

func TestLogger_WithComponent(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	newLogger := logger.WithComponent("test")
	if newLogger == nil {
		t.Fatal("WithComponent returned nil")
	}
}

func TestLogger_WithError(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	newLogger := logger.WithError(nil)
	if newLogger == nil {
		t.Fatal("WithError returned nil")
	}
}

func TestLogger_LogLevels(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	// Should not panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
}

func TestLogger_LogLevelsWithFields(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	// Should not panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
}

func TestLogger_LogFormatted(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	// Should not panic
	logger.Debugf("debug message: %s", "test")
	logger.Infof("info message: %s", "test")
	logger.Warnf("warn message: %s", "test")
	logger.Errorf("error message: %s", "test")
}

func TestLogger_Sync(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	err = logger.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}

func TestLogger_Close(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	err = logger.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestGlobalLogger(t *testing.T) {
	logger := GetGlobalLogger()
	if logger == nil {
		t.Fatal("Global logger is nil")
	}
}

func TestSetGlobalLogger(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	SetGlobalLogger(logger)
	
	globalLogger := GetGlobalLogger()
	if globalLogger != logger {
		t.Error("Global logger was not set correctly")
	}
}

func TestInitGlobalLogger(t *testing.T) {
	err := InitGlobalLogger("info", false)
	if err != nil {
		t.Errorf("InitGlobalLogger failed: %v", err)
	}
	
	logger := GetGlobalLogger()
	if logger == nil {
		t.Fatal("Global logger is nil after init")
	}
}

func TestStdoutWriter_Write(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	writer := NewStdoutWriter(logger, 0)
	
	n, err := writer.Write([]byte("test message"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	
	if n != len("test message") {
		t.Errorf("Expected %d bytes written, got %d", len("test message"), n)
	}
}

func TestStderrWriter_Write(t *testing.T) {
	logger, err := NewLogger("info", false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	writer := NewStderrWriter(logger, 0)
	
	n, err := writer.Write([]byte("test message"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	
	if n != len("test message") {
		t.Errorf("Expected %d bytes written, got %d", len("test message"), n)
	}
}
