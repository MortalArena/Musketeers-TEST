package agent_bridge

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestServer_Start(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)
	mb := NewMultiplexedBridge(log)

	// استخدام منفذ عشوائي
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	// ✅ تمرير nil للعقدة في الاختبار
	server := NewServer(nil, addr, sm, mb, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Server.Start failed: %v", err)
	}

	// إعطاء الخادم وقتاً للبدء
	time.Sleep(100 * time.Millisecond)

	err = server.Stop()
	if err != nil {
		t.Fatalf("Server.Stop failed: %v", err)
	}
}

func TestServer_Start_AlreadyRunning(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)
	mb := NewMultiplexedBridge(log)

	// استخدام منفذ عشوائي
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	// ✅ تمرير nil للعقدة في الاختبار
	server := NewServer(nil, addr, sm, mb, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Server.Start failed: %v", err)
	}

	// محاولة البدء مرة أخرى
	err = server.Start(ctx)
	if err == nil {
		t.Fatal("Expected error when server is already running")
	}

	server.Stop()
}

func TestServer_Stop(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)
	mb := NewMultiplexedBridge(log)

	// استخدام منفذ عشوائي
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	// ✅ تمرير nil للعقدة في الاختبار
	server := NewServer(nil, addr, sm, mb, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Server.Start failed: %v", err)
	}

	// إعطاء الخادم وقتاً للبدء
	time.Sleep(100 * time.Millisecond)

	err = server.Stop()
	if err != nil {
		t.Fatalf("Server.Stop failed: %v", err)
	}
}

func TestServer_Stop_NotRunning(t *testing.T) {
	log := logrus.New()
	sm := NewSessionManager(log)
	mb := NewMultiplexedBridge(log)

	// ✅ تمرير nil للعقدة في الاختبار
	server := NewServer(nil, "127.0.0.1:5001", sm, mb, log)

	err := server.Stop()
	// Stop لا يرجع خطأ إذا لم يكن الخادم يعمل
	if err != nil {
		t.Fatalf("Server.Stop failed: %v", err)
	}
}
