package gateway

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/MortalArena/Musketeers/pkg/protocol"
)

// NormalizeSitePath يطبّع مسار ملف الموقع ويرفض path traversal
func NormalizeSitePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	if p == "" || p == "/" {
		return "index.html", nil
	}
	p = strings.TrimPrefix(p, "/")

	clean := path.Clean(p)
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", fmt.Errorf("مسار غير مسموح")
	}
	if strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("مسار مطلق غير مسموح")
	}
	return clean, nil
}

// ParseManifest يفك manifest.json مع حدود أمان
func ParseManifest(data []byte) (*protocol.SiteManifest, error) {
	if len(data) > protocol.MaxManifestSize {
		return nil, fmt.Errorf("المانيفست كبير جداً")
	}
	var m protocol.SiteManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if len(m.Files) > protocol.MaxManifestFiles {
		return nil, fmt.Errorf("عدد الملفات يتجاوز الحد")
	}
	for filePath := range m.Files {
		if _, err := NormalizeSitePath(filePath); err != nil {
			return nil, fmt.Errorf("مسار غير صالح في المانيفست: %s", filePath)
		}
	}
	return &m, nil
}

// ResolveCID يحل مساراً إلى CID من المانيفست
func ResolveCID(manifest *protocol.SiteManifest, requestPath string) (string, string, error) {
	normalized, err := NormalizeSitePath(requestPath)
	if err != nil {
		return "", "", err
	}
	cid, ok := manifest.Files[normalized]
	if !ok {
		// محاولة index.html للمجلدات
		if !strings.HasSuffix(normalized, "index.html") {
			cid, ok = manifest.Files[normalized+"/index.html"]
			if ok {
				normalized = normalized + "/index.html"
			}
		}
	}
	if !ok {
		return "", "", fmt.Errorf("ملف غير موجود: %s", normalized)
	}
	return cid, normalized, nil
}

// ContentType يخمّن نوع المحتوى من الامتداد
func ContentType(filePath string) string {
	switch {
	case strings.HasSuffix(filePath, ".html"), strings.HasSuffix(filePath, ".htm"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(filePath, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(filePath, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(filePath, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(filePath, ".png"):
		return "image/png"
	case strings.HasSuffix(filePath, ".jpg"), strings.HasSuffix(filePath, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(filePath, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(filePath, ".wasm"):
		return "application/wasm"
	default:
		return "application/octet-stream"
	}
}
