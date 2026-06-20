package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

// MaxMessageSize الحد الأقصى لحجم الرسالة (10MB)
const MaxMessageSize = 10 * 1024 * 1024

// MessageType نوع الرسالة
type MessageType string

const (
	MessageTypeTaskRequest  MessageType = "task_request"
	MessageTypeTaskResponse MessageType = "task_response"
	MessageTypeHeartbeat    MessageType = "heartbeat"
	MessageTypeHeartbeatAck MessageType = "heartbeat_ack"
	MessageTypeError        MessageType = "error"
)

// Message رسالة بروتوكول
type Message struct {
	Type MessageType `json:"type"`
	Data []byte      `json:"data"`
}

// NewMessage ينشئ رسالة جديدة
func NewMessage(msgType MessageType, data []byte) *Message {
	return &Message{
		Type: msgType,
		Data: data,
	}
}

// WriteMessage يكتب رسالة إلى اتصال
func WriteMessage(conn net.Conn, msg *Message) error {
	// ترميز الرسالة كـ JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// كتابة طول الرسالة أولاً (4 bytes)
	length := uint32(len(data))
	if err := binary.Write(conn, binary.BigEndian, length); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	// كتابة الرسالة
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to write message data: %w", err)
	}

	return nil
}

// ReadMessage يقرأ رسالة من اتصال
func ReadMessage(conn net.Conn) (*Message, error) {
	// قراءة طول الرسالة (4 bytes)
	var length uint32
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}

	// فحص الحد الأقصى لحجم الرسالة
	if length == 0 || length > MaxMessageSize {
		return nil, fmt.Errorf("invalid message length: %d (max: %d)", length, MaxMessageSize)
	}

	// قراءة الرسالة
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, fmt.Errorf("failed to read message data: %w", err)
	}

	// فك ترميز الرسالة
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}
