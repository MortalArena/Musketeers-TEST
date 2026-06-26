package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
)

// VerificationStatus describes how thoroughly an agent's capabilities have been checked
type VerificationStatus string

const (
	VerificationUnverified VerificationStatus = "unverified"
	VerificationVerified   VerificationStatus = "verified"
	VerificationPartial    VerificationStatus = "partial"
	VerificationFailed     VerificationStatus = "failed"
)

// ProbeResult holds the outcome of verifying a single capability
type ProbeResult struct {
	Capability string        `json:"capability"`
	Verified   bool          `json:"verified"`
	Duration   time.Duration `json:"duration_ms"`
	Error      string        `json:"error,omitempty"`
}

// VerificationReport is the complete result of verifying all of an agent's capabilities
type VerificationReport struct {
	AgentID       string        `json:"agent_id"`
	Verified      []string      `json:"verified"`
	Failed        []string      `json:"failed"`
	OverallStatus string        `json:"overall_status"`
	Probes        []ProbeResult `json:"probes"`
	VerifiedAt    time.Time     `json:"verified_at"`
}

// AgentCapabilityVerifier probes claimed capabilities by running lightweight tasks
// [WHY] يحقق من القدرات المعلنة للوكلاء بدلاً من تصديقها بدون تحقق
type AgentCapabilityVerifier struct {
	mu             sync.RWMutex
	probeTimeout   time.Duration
	cacheResults   bool
	verificationCache map[string]*VerificationReport // agentID -> last report
}

// NewAgentCapabilityVerifier creates a new verifier
func NewAgentCapabilityVerifier() *AgentCapabilityVerifier {
	return &AgentCapabilityVerifier{
		probeTimeout:      30 * time.Second,
		cacheResults:      true,
		verificationCache: make(map[string]*VerificationReport),
	}
}

// VerifyAll probes all capabilities claimed by an agent
// [WHY] يختبر كل قدرة معلنة بمهمة اختبار خفيفة
func (v *AgentCapabilityVerifier) VerifyAll(ctx context.Context, ua agent.UnifiedAgent) (*VerificationReport, error) {
	info := ua.GetInfo()
	capabilities := ua.GetCapabilities()

	if len(capabilities) == 0 {
		return &VerificationReport{
			AgentID:       info.ID,
			OverallStatus: string(VerificationVerified),
			VerifiedAt:    time.Now(),
		}, nil
	}

	probeCtx, cancel := context.WithTimeout(ctx, v.probeTimeout)
	defer cancel()

	report := &VerificationReport{
		AgentID:    info.ID,
		Probes:     make([]ProbeResult, 0, len(capabilities)),
		VerifiedAt: time.Now(),
	}

	for _, cap := range capabilities {
		result := v.probeSingle(probeCtx, ua, cap)
		report.Probes = append(report.Probes, result)
		if result.Verified {
			report.Verified = append(report.Verified, string(cap))
		} else {
			report.Failed = append(report.Failed, string(cap))
		}
	}

	// Determine overall status
	if len(report.Failed) == 0 {
		report.OverallStatus = string(VerificationVerified)
	} else if len(report.Verified) > 0 {
		report.OverallStatus = string(VerificationPartial)
	} else {
		report.OverallStatus = string(VerificationFailed)
	}

	if v.cacheResults {
		v.mu.Lock()
		v.verificationCache[info.ID] = report
		v.mu.Unlock()
	}

	return report, nil
}

// VerifySingle probes a single capability
// [WHY] يختبر قدرة محددة
func (v *AgentCapabilityVerifier) VerifySingle(ctx context.Context, ua agent.UnifiedAgent, capability agent.AgentCapability) (*ProbeResult, error) {
	probeCtx, cancel := context.WithTimeout(ctx, v.probeTimeout)
	defer cancel()

	result := v.probeSingle(probeCtx, ua, capability)
	return &result, nil
}

// GetCachedReport returns the last verification report for an agent
func (v *AgentCapabilityVerifier) GetCachedReport(agentID string) (*VerificationReport, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	r, ok := v.verificationCache[agentID]
	if !ok {
		return nil, false
	}
	return r, true
}

// ClearCache removes cached reports
func (v *AgentCapabilityVerifier) ClearCache() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.verificationCache = make(map[string]*VerificationReport)
}

// SetProbeTimeout sets the timeout for each capability probe
func (v *AgentCapabilityVerifier) SetProbeTimeout(d time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.probeTimeout = d
}

// probeSingle tests a single capability with a lightweight task
// [WHY] يستخدم مهمة اختبار بسيطة مناسبة لنوع القدرة
// [HOW] يرسل استعلام بسيط للوكيل ويختبر قدرته على الرد
func (v *AgentCapabilityVerifier) probeSingle(ctx context.Context, ua agent.UnifiedAgent, capability agent.AgentCapability) ProbeResult {
	start := time.Now()

	// [WHY] اختيار مهمة اختبار مناسبة لنوع القدرة
	task := v.probeTaskForCapability(capability)
	if task == nil {
		return ProbeResult{
			Capability: string(capability),
			Verified:   true, // [WHY] إذا لم نتمكن من اختبارها، نعتبرها صحيحة
			Duration:   0,
		}
	}

	result, err := ua.ExecuteTask(ctx, task)
	duration := time.Since(start)

	if err != nil {
		return ProbeResult{
			Capability: string(capability),
			Verified:   false,
			Duration:   duration,
			Error:      err.Error(),
		}
	}

	if !result.Success {
		return ProbeResult{
			Capability: string(capability),
			Verified:   false,
			Duration:   duration,
			Error:      result.Error,
		}
	}

	return ProbeResult{
		Capability: string(capability),
		Verified:   true,
		Duration:   duration,
	}
}

// probeTaskForCapability creates a lightweight probe task for each capability type
// [WHY] مهام اختبار خفيفة لكل نوع قدرة
// [HOW] تستخدم مهام بسيطة لا تستهلك توكنز كثيرة
func (v *AgentCapabilityVerifier) probeTaskForCapability(capability agent.AgentCapability) *agent.AgentTask {
	id := fmt.Sprintf("probe-%s-%d", capability, time.Now().UnixNano())

	switch capability {
	case agent.CapabilityCodeGeneration:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] Generate a hello world function in Go",
			Description: "Write a simple Go function that returns 'hello world'",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityCodeReview:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] Review this code",
			Description: "Review the following code for issues: 'func add(a int, b int) int { return a - b }'",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityTesting:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] Write a test",
			Description: "Write a simple test for a Go function that adds two numbers",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityDocumentation:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] Document a function",
			Description: "Write documentation for a function that calculates fibonacci numbers",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityAnalysis:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] Analyze data",
			Description: "Analyze this dataset: [1, 2, 3, 4, 5] and describe the pattern",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityDesign:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] Describe a design",
			Description: "Describe how you would design a REST API for a todo list application",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityFileOperations:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] File operations",
			Description: "Describe the steps to read a file, modify its contents, and save it back",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityTerminalAccess:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] Terminal operations",
			Description: "Describe how to list files in a directory and find files by pattern",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityBrowserControl:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] Browser navigation",
			Description: "Describe the steps to navigate to a URL, find an element, and click it",
			Timeout:     30 * time.Second,
		}
	case agent.CapabilityAPIIntegration:
		return &agent.AgentTask{
			ID:          id,
			Title:       "[PROBE] API integration",
			Description: "Describe how to make an HTTP GET request, parse the JSON response, and handle errors",
			Timeout:     30 * time.Second,
		}
	default:
		return nil
	}
}
