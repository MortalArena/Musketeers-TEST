package crypto

import (
	"fmt"

	"github.com/hashicorp/vault/shamir"
)

const (
	TotalShares    = 3 // Total number of shares
	RequiredShares = 2 // Number required for reconstruction
)

// SplitMasterKey splits master key into encrypted shares
func SplitMasterKey(masterKey []byte) ([][]byte, error) {
	if len(masterKey) == 0 {
		return nil, fmt.Errorf("master key cannot be empty")
	}

	shares, err := shamir.Split(masterKey, TotalShares, RequiredShares)
	if err != nil {
		return nil, fmt.Errorf("failed to split master key: %w", err)
	}

	return shares, nil
}

// ReconstructMasterKey reconstructs master key from available shares
func ReconstructMasterKey(shares [][]byte) ([]byte, error) {
	if len(shares) < RequiredShares {
		return nil, fmt.Errorf("insufficient shares: need at least %d, got %d", RequiredShares, len(shares))
	}

	reconstructed, err := shamir.Combine(shares)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct master key: %w", err)
	}

	return reconstructed, nil
}
