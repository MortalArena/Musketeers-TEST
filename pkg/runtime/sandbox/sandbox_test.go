package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessSandboxRun(t *testing.T) {
	if err := os.MkdirAll(filepath.Join(t.TempDir(), "bin"), 0700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	s := NewProcessSandbox()
	out, err := s.Run("echo hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if string(out) != "hello\n" && string(out) != "hello\r\n" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestProcessSandboxRejectsForbiddenCommand(t *testing.T) {
	s := NewProcessSandbox()
	s.SetAllowedCommands([]string{"echo"})
	if _, err := s.Run("rm -rf /"); err == nil {
		t.Fatal("expected forbidden command error")
	}
}
