package security

import (
	"crypto/tls"
	"os"
	"testing"
)

func TestTLSConfigBuilder(t *testing.T) {
	// توليد شهادات للتطوير
	certFile := "C:/tmp/test.crt"
	keyFile := "C:/tmp/test.key"
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	err := GenerateSelfSignedCert(certFile, keyFile, []string{"localhost", "127.0.0.1"})
	if err != nil {
		t.Fatalf("فشل توليد الشهادات: %v", err)
	}

	// اختبار البناء
	builder := NewTLSConfigBuilder().
		WithCertFiles(certFile, keyFile)

	config, err := builder.Build()
	if err != nil {
		t.Fatalf("فشل بناء TLS config: %v", err)
	}

	// التحقق من الإعدادات
	if config.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion يجب أن يكون TLS 1.3")
	}

	if config.Renegotiation != tls.RenegotiateNever {
		t.Errorf("Renegotiation يجب أن يكون Never")
	}

	if len(config.CipherSuites) != 3 {
		t.Errorf("يجب أن يكون 3 cipher suites")
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	certFile := "C:/tmp/test2.crt"
	keyFile := "C:/tmp/test2.key"
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	err := GenerateSelfSignedCert(certFile, keyFile, []string{"localhost", "127.0.0.1"})
	if err != nil {
		t.Fatalf("فشل توليد الشهادات: %v", err)
	}

	// التحقق من وجود الملفات
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		t.Error("ملف الشهادة غير موجود")
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Error("ملف المفتاح غير موجود")
	}
}
