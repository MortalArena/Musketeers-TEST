package crypto

import (
	"fmt"

	"github.com/hashicorp/vault/shamir"
)

const (
	TotalShares    = 3
	RequiredShares = 2
)

// SplitMasterKey يقسم المفتاح الرئيسي إلى أجزاء مشفرة
func SplitMasterKey(masterKey []byte) ([][]byte, error) {
	if len(masterKey) == 0 {
		return nil, fmt.Errorf("master key cannot be empty")
	}

	// استخدام مكتبة Hashicorp Shamir الموثوقة
	shares, err := shamir.Split(masterKey, TotalShares, RequiredShares)
	if err != nil {
		return nil, fmt.Errorf("failed to split master key: %w", err)
	}

	return shares, nil
}

// ReconstructMasterKey يعيد بناء المفتاح الرئيسي من الأجزاء المتاحة
func ReconstructMasterKey(shares [][]byte) ([]byte, error) {
	if len(shares) < RequiredShares {
		return nil, fmt.Errorf("insufficient shares: need at least %d, got %d", RequiredShares, len(shares))
	}

	// إعادة البناء
	reconstructed, err := shamir.Combine(shares)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct master key: %w", err)
	}

	return reconstructed, nil
}
