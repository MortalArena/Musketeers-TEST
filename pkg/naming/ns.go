package naming

import (
	"fmt"
	"strings"

	"golang.org/x/text/unicode/norm"
)

const (
	DomainSuffix    = ".mskt"
	GatewayDomain   = "mskt.run"
	MaxDomainLength = 63
	MaxProviders    = 32
)

// NormalizeDomainName يطبّع اسم النطاق (NFC + lowercase ASCII)
func NormalizeDomainName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = norm.NFC.String(name)
	name = strings.ToLower(name)

	if !strings.HasSuffix(name, DomainSuffix) {
		return "", fmt.Errorf("النطاق يجب أن ينتهي بـ %s", DomainSuffix)
	}

	label := strings.TrimSuffix(name, DomainSuffix)
	if label == "" {
		return "", fmt.Errorf("اسم النطاق فارغ")
	}
	if len(label) > MaxDomainLength {
		return "", fmt.Errorf("اسم النطاق طويل جداً (الحد %d)", MaxDomainLength)
	}

	// رفض mixed-script (حروف من أبجديات مختلفة)
	if hasMixedScript(label) {
		return "", fmt.Errorf("النطاق يحتوي على أبجديات مختلطة")
	}

	// أحرف مسموحة: a-z, 0-9, hyphen (لا يبدأ أو ينتهي بـ -)
	for i, r := range label {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' {
			if i == 0 || i == len(label)-1 {
				return "", fmt.Errorf("النطاق لا يمكن أن يبدأ أو ينتهي بشرطة")
			}
			continue
		}
		return "", fmt.Errorf("حرف غير مسموح في النطاق: %c", r)
	}

	return name, nil
}

// hasMixedScript يكتشف أحرف غير ASCII (حماية homograph — النطاقات ASCII فقط)
func hasMixedScript(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

// DHTKeyPrefix بادئة مفاتيح DHT للنطاقات
const DHTKeyPrefix = "/mskt/domain/"

// DHTKey يبني مفتاح DHT للنطاق
func DHTKey(name string) string {
	return DHTKeyPrefix + name
}
