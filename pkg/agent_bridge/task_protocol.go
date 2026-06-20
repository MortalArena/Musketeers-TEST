package agent_bridge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent_bridge/protocol"
)

// TaskRequest طلب مهمة من Studio إلى Agent
type TaskRequest struct {
	TaskID     string                 `json:"task_id"`
	Type       string                 `json:"type"` // "execute", "query", "file_upload", "file_download"
	Payload    map[string]interface{} `json:"payload"`
	Priority   int                    `json:"priority"` // 0=low, 1=normal, 2=high, 3=emergency
	CreatedAt  time.Time              `json:"created_at"`
	TimeoutSec int                    `json:"timeout_sec"` // مهلة التنفيذ بالثواني
}

// TaskResponse استجابة مهمة من Agent إلى Studio
type TaskResponse struct {
	TaskID      string                 `json:"task_id"`
	Success     bool                   `json:"success"`
	Result      map[string]interface{} `json:"result"`
	Error       string                 `json:"error,omitempty"`
	CompletedAt time.Time              `json:"completed_at"`
}

// TaskProtocol يدير بروتوكول المهام
type TaskProtocol struct{}

// NewTaskProtocol ينشئ بروتوكول مهام جديد
func NewTaskProtocol() *TaskProtocol {
	return &TaskProtocol{}
}

// CreateTaskRequest ينشئ طلب مهمة جديد
func (tp *TaskProtocol) CreateTaskRequest(taskType string, payload map[string]interface{}, priority int, timeoutSec int) (*TaskRequest, error) {
	taskID, err := generateTaskID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate task ID: %w", err)
	}

	return &TaskRequest{
		TaskID:     taskID,
		Type:       taskType,
		Payload:    payload,
		Priority:   priority,
		CreatedAt:  time.Now(),
		TimeoutSec: timeoutSec,
	}, nil
}

// CreateTaskResponse ينشئ استجابة مهمة
func (tp *TaskProtocol) CreateTaskResponse(taskID string, success bool, result map[string]interface{}, errMsg string) *TaskResponse {
	return &TaskResponse{
		TaskID:      taskID,
		Success:     success,
		Result:      result,
		Error:       errMsg,
		CompletedAt: time.Now(),
	}
}

// EncodeTaskRequest يرمز طلب مهمة إلى رسالة بروتوكول
func (tp *TaskProtocol) EncodeTaskRequest(req *TaskRequest) (*protocol.Message, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task request: %w", err)
	}

	return protocol.NewMessage(protocol.MessageTypeTaskRequest, data), nil
}

// DecodeTaskRequest يفك ترميز طلب مهمة من رسالة بروتوكول
func (tp *TaskProtocol) DecodeTaskRequest(msg *protocol.Message) (*TaskRequest, error) {
	if msg.Type != protocol.MessageTypeTaskRequest {
		return nil, fmt.Errorf("invalid message type: expected %s, got %s", protocol.MessageTypeTaskRequest, msg.Type)
	}

	var req TaskRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task request: %w", err)
	}

	return &req, nil
}

// EncodeTaskResponse يرمز استجابة مهمة إلى رسالة بروتوكول
func (tp *TaskProtocol) EncodeTaskResponse(resp *TaskResponse) (*protocol.Message, error) {
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task response: %w", err)
	}

	return protocol.NewMessage(protocol.MessageTypeTaskResponse, data), nil
}

// DecodeTaskResponse يفك ترميز استجابة مهمة من رسالة بروتوكول
func (tp *TaskProtocol) DecodeTaskResponse(msg *protocol.Message) (*TaskResponse, error) {
	if msg.Type != protocol.MessageTypeTaskResponse {
		return nil, fmt.Errorf("invalid message type: expected %s, got %s", protocol.MessageTypeTaskResponse, msg.Type)
	}

	var resp TaskResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task response: %w", err)
	}

	return &resp, nil
}

// ValidateTaskRequest يتحقق من صحة طلب مهمة
func (tp *TaskProtocol) ValidateTaskRequest(req *TaskRequest) error {
	if req.TaskID == "" {
		return fmt.Errorf("task ID is required")
	}
	if req.Type == "" {
		return fmt.Errorf("task type is required")
	}
	if req.Priority < 0 || req.Priority > 3 {
		return fmt.Errorf("invalid priority: must be 0-3")
	}
	if req.TimeoutSec <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

// ValidateTaskResponse يتحقق من صحة استجابة مهمة
func (tp *TaskProtocol) ValidateTaskResponse(resp *TaskResponse) error {
	if resp.TaskID == "" {
		return fmt.Errorf("task ID is required")
	}
	if !resp.Success && resp.Error == "" {
		return fmt.Errorf("error message required when success is false")
	}
	return nil
}

// generateTaskID يولد معرف مهمة فريد
func generateTaskID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "task_" + hex.EncodeToString(b), nil
}

// [SAFETY] generateSessionID يولد معرف جلسة فريد بشكل آمن
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// [FALLBACK] If crypto/rand fails, use timestamp (less secure but better than panic)
		return "session_" + fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return "session_" + hex.EncodeToString(b)
}
