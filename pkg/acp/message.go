package acp

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
)

const (
	ProtocolVersion = "acp/v1"
	ProtocolID      = "/nr/acp/1.0.0"

	IntentTaskRequest  = "task.request"
	IntentTaskResponse = "task.response"
	IntentTaskError    = "task.error"

	TaskEcho      = "echo"
	TaskPing      = "ping"
	TaskTranslate = "translate"
	TaskExecute   = "task.execute"
)

// Envelope رسالة ACP موحّدة
type Envelope struct {
	Protocol  string          `json:"protocol"`
	Intent    string          `json:"intent"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Task      string          `json:"task,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Output    json.RawMessage `json:"output,omitempty"`
	Error     string          `json:"error,omitempty"`
	Callback  string          `json:"callback,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Timestamp int64           `json:"timestamp"`
	Signature string          `json:"signature"`
}

// EnvelopePayload payload ثابت للتوقيع
func EnvelopePayload(env *Envelope) string {
	return strings.Join([]string{
		env.Protocol,
		env.Intent,
		env.From,
		env.To,
		env.Task,
		string(env.Input),
		string(env.Output),
		env.Error,
		env.Callback,
		env.RequestID,
		strconv.FormatInt(env.Timestamp, 10),
	}, "|")
}

// SignEnvelope يوقّع الرسالة
func SignEnvelope(env *Envelope, priv ed25519.PrivateKey) error {
	if env.Timestamp == 0 {
		env.Timestamp = time.Now().Unix()
	}
	if env.Protocol == "" {
		env.Protocol = ProtocolVersion
	}
	payload := EnvelopePayload(env)
	sig, err := nrcrypto.SignPayloadHex(priv, nrcrypto.DomainACP, payload)
	if err != nil {
		return err
	}
	env.Signature = sig
	return nil
}

// VerifyEnvelope يتحقق من التوقيع
func VerifyEnvelope(env *Envelope, pub ed25519.PublicKey) error {
	if env.Protocol != ProtocolVersion {
		return fmt.Errorf("إصدار بروتوكول غير مدعوم: %s", env.Protocol)
	}
	if env.Signature == "" {
		return fmt.Errorf("توقيع مطلوب")
	}
	payload := EnvelopePayload(env)
	return nrcrypto.VerifyPayloadHex(pub, nrcrypto.DomainACP, payload, env.Signature)
}

// NewTaskRequest ينشئ طلب مهمة
func NewTaskRequest(from, to, task string, input json.RawMessage, callback, requestID string, priv ed25519.PrivateKey) (*Envelope, error) {
	env := &Envelope{
		Protocol:  ProtocolVersion,
		Intent:    IntentTaskRequest,
		From:      from,
		To:        to,
		Task:      task,
		Input:     input,
		Callback:  callback,
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
	}
	if err := SignEnvelope(env, priv); err != nil {
		return nil, err
	}
	return env, nil
}

// NewTaskResponse ينشئ استجابة مهمة
func NewTaskResponse(from, to, task, requestID string, output json.RawMessage, priv ed25519.PrivateKey) (*Envelope, error) {
	env := &Envelope{
		Protocol:  ProtocolVersion,
		Intent:    IntentTaskResponse,
		From:      from,
		To:        to,
		Task:      task,
		Output:    output,
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
	}
	if err := SignEnvelope(env, priv); err != nil {
		return nil, err
	}
	return env, nil
}

// NewTaskError ينشئ رسالة خطأ
func NewTaskError(from, to, task, requestID, errMsg string, priv ed25519.PrivateKey) (*Envelope, error) {
	env := &Envelope{
		Protocol:  ProtocolVersion,
		Intent:    IntentTaskError,
		From:      from,
		To:        to,
		Task:      task,
		Error:     errMsg,
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
	}
	if err := SignEnvelope(env, priv); err != nil {
		return nil, err
	}
	return env, nil
}
