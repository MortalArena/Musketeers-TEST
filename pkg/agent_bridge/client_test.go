package agent_bridge

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestClient_NewClient(t *testing.T) {
	log := logrus.New()
	client := NewClient("127.0.0.1:5001", log)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

func TestClient_IsConnected(t *testing.T) {
	log := logrus.New()
	client := NewClient("127.0.0.1:5001", log)

	if client.IsConnected() {
		t.Fatal("Expected false for IsConnected before Connect")
	}
}

func TestClient_Connect_NotConnected(t *testing.T) {
	log := logrus.New()
	client := NewClient("127.0.0.1:5001", log)

	ctx := context.Background()
	err := client.Connect(ctx)
	// من المتوقع أن يفشل لأنه لا يوجد خادم يعمل
	if err == nil {
		t.Fatal("Expected error when connecting to non-existent server")
	}
}

func TestClient_Disconnect_NotConnected(t *testing.T) {
	log := logrus.New()
	client := NewClient("127.0.0.1:5001", log)

	err := client.Disconnect()
	// Disconnect لا يرجع خطأ إذا لم يكن متصلاً
	if err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
}
