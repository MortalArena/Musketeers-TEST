package gateway

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/sirupsen/logrus"
)

// Server بوابة HTTP لمواقع .mskt
type Server struct {
	node   *node.Node
	log    *logrus.Logger
	server *http.Server
}

// NewServer ينشئ بوابة HTTP
// المسارات: /d/{domain}.mskt/{path} — عزل لكل نطاق (يمنع XSS cross-domain)
func NewServer(n *node.Node, port int, log *logrus.Logger) *Server {
	s := &Server{node: n, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("/d/", s.handleSite)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", s.handleIndex)

	s.server = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	return s
}

// Start يبدأ البوابة
func (s *Server) Start() error {
	s.log.WithField("addr", s.server.Addr).Info("بدء HTTP Gateway")
	return s.server.ListenAndServe()
}

// StartTLS يبدأ البوابة مع TLS
func (s *Server) StartTLS(certFile, keyFile string) error {
	if certFile == "" || keyFile == "" {
		s.log.Info("توليد شهادة TLS ذاتية التوقيع في الذاكرة...")
		cert, err := generateSelfSignedCert()
		if err != nil {
			return fmt.Errorf("فشل توليد شهادة TLS: %w", err)
		}
		s.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		s.log.WithField("addr", s.server.Addr).Info("بدء HTTPS Gateway (TLS self-signed)")
		return s.server.ListenAndServeTLS("", "")
	}
	s.log.WithFields(logrus.Fields{
		"addr": s.server.Addr,
		"cert": certFile,
		"key":  keyFile,
	}).Info("بدء HTTPS Gateway (TLS)")
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Musketeers"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// Stop يوقف البوابة
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Musketeers Gateway</title></head>
<body><h1>Musketeers Gateway</h1>
<p>استخدم: <code>/d/example.mskt/</code> لزيارة موقع .mskt</p>
</body></html>`)
}

func (s *Server) handleSite(w http.ResponseWriter, r *http.Request) {
	// /d/example.mskt/path/to/file
	rest := strings.TrimPrefix(r.URL.Path, "/d/")
	if rest == "" {
		http.Error(w, "اسم النطاق مطلوب", http.StatusBadRequest)
		return
	}

	parts := strings.SplitN(rest, "/", 2)
	domain := parts[0]
	if !strings.HasSuffix(domain, ".mskt") {
		http.Error(w, "Invalid domain — must end with .mskt", http.StatusBadRequest)
		return
	}

	filePath := "/"
	if len(parts) == 2 {
		filePath = parts[1]
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	rec, err := s.node.ResolveDomain(ctx, domain)
	if err != nil {
		http.Error(w, "النطاق غير موجود: "+err.Error(), http.StatusNotFound)
		return
	}
	if rec.ManifestCID == "" {
		http.Error(w, "النطاق لا يحتوي على موقع (ManifestCID فارغ)", http.StatusNotFound)
		return
	}

	manifestData, err := s.node.FetchContent(ctx, rec.ManifestCID)
	if err != nil {
		http.Error(w, "فشل جلب المانيفست: "+err.Error(), http.StatusBadGateway)
		return
	}

	manifest, err := ParseManifest(manifestData)
	if err != nil {
		http.Error(w, "مانيفست غير صالح: "+err.Error(), http.StatusBadGateway)
		return
	}

	cid, resolvedPath, err := ResolveCID(manifest, filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	data, err := s.node.FetchContent(ctx, cid)
	if err != nil {
		http.Error(w, "فشل جلب المحتوى: "+err.Error(), http.StatusBadGateway)
		return
	}

	// عزل الأصل — كل نطاق في مسار منفصل
	w.Header().Set("Content-Type", ContentType(resolvedPath))
	w.Header().Set("X-NR-Domain", domain)
	w.Header().Set("X-NR-CID", cid)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
	w.Write(data)
}
