package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// TLSConfigBuilder يبني إعدادات TLS آمنة
type TLSConfigBuilder struct {
	certFile       string
	keyFile        string
	caFile         string
	minVersion     uint16
	cipherSuites   []uint16
	curvePrefs     []tls.CurveID
	clientAuth     tls.ClientAuthType
	enableHSTS     bool
	enableOCSP     bool
	sessionTimeout time.Duration
}

// NewTLSConfigBuilder ينشئ builder جديد
func NewTLSConfigBuilder() *TLSConfigBuilder {
	return &TLSConfigBuilder{
		minVersion: tls.VersionTLS13,
		cipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		curvePrefs: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		clientAuth:     tls.NoClientCert,
		enableHSTS:     true,
		sessionTimeout: 24 * time.Hour,
	}
}

// WithCertFiles يحدد ملفات الشهادات
func (b *TLSConfigBuilder) WithCertFiles(certFile, keyFile string) *TLSConfigBuilder {
	b.certFile = certFile
	b.keyFile = keyFile
	return b
}

// WithCAFile يحدد ملف CA
func (b *TLSConfigBuilder) WithCAFile(caFile string) *TLSConfigBuilder {
	b.caFile = caFile
	return b
}

// WithClientAuth يحدد نوع مصادقة العميل
func (b *TLSConfigBuilder) WithClientAuth(auth tls.ClientAuthType) *TLSConfigBuilder {
	b.clientAuth = auth
	return b
}

// Build يبني TLS Config
func (b *TLSConfigBuilder) Build() (*tls.Config, error) {
	// تحميل الشهادات
	cert, err := tls.LoadX509KeyPair(b.certFile, b.keyFile)
	if err != nil {
		return nil, fmt.Errorf("فشل تحميل الشهادات: %w", err)
	}

	config := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               b.minVersion,
		MaxVersion:               tls.VersionTLS13,
		CipherSuites:             b.cipherSuites,
		CurvePreferences:         b.curvePrefs,
		PreferServerCipherSuites: true,
		ClientAuth:               b.clientAuth,
		SessionTicketsDisabled:   false,
		Renegotiation:            tls.RenegotiateNever, // منع downgrade attacks
	}

	// إضافة CA pool إذا وُجد
	if b.caFile != "" {
		caCert, err := os.ReadFile(b.caFile)
		if err == nil {
			certPool := x509.NewCertPool()
			certPool.AppendCertsFromPEM(caCert)
			config.ClientCAs = certPool
		}
	}

	return config, nil
}

// GenerateSelfSignedCert يولد شهادة ذاتية للتطوير
func GenerateSelfSignedCert(certFile, keyFile string, hosts []string) error {
	// إنشاء المفتاح الخاص
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("فشل توليد المفتاح: %w", err)
	}

	// إنشاء serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("فشل توليد serial: %w", err)
	}

	// إنشاء template الشهادة
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Musketeers Dev"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// إضافة SANs
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// إنشاء الشهادة
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("فشل إنشاء الشهادة: %w", err)
	}

	// إنشاء المجلدات
	os.MkdirAll(filepath.Dir(certFile), 0755)
	os.MkdirAll(filepath.Dir(keyFile), 0755)

	// حفظ الشهادة
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("فشل إنشاء ملف الشهادة: %w", err)
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// حفظ المفتاح
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("فشل إنشاء ملف المفتاح: %w", err)
	}
	defer keyOut.Close()
	privBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("فشل تحويل المفتاح: %w", err)
	}
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	return nil
}
