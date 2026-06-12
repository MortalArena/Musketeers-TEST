package sandbox

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

type Sandbox interface {
	Run(command string) ([]byte, error)
	SetAllowedCommands(commands []string)
}

type ProcessSandbox struct {
	mu              sync.RWMutex
	allowedCommands []string
}

func NewProcessSandbox() *ProcessSandbox {
	return &ProcessSandbox{}
}

func (s *ProcessSandbox) SetAllowedCommands(commands []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowedCommands = append([]string(nil), commands...)
}

func (s *ProcessSandbox) Run(command string) ([]byte, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("command is empty")
	}
	if !s.isAllowed(command) {
		return nil, fmt.Errorf("command is not allowed: %s", command)
	}
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", command).CombinedOutput()
	}
	return exec.Command("sh", "-c", command).CombinedOutput()
}

func (s *ProcessSandbox) isAllowed(command string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.allowedCommands) == 0 {
		return true
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	base := strings.ToLower(fields[0])
	for _, allowed := range s.allowedCommands {
		if strings.ToLower(allowed) == base {
			return true
		}
	}
	return false
}
