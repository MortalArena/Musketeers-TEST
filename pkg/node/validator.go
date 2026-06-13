package node

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/identity"
	"github.com/MortalArena/Musketeers/pkg/naming"
	"github.com/MortalArena/Musketeers/pkg/search"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DHTValidators يوفّر validators لسجلات DHT
type DHTValidators struct {
	founderPub ed25519.PublicKey
	crl        *identity.CRLCache
}

// NewDHTValidators ينشئ validators
func NewDHTValidators(founderPub ed25519.PublicKey, crl *identity.CRLCache) *DHTValidators {
	return &DHTValidators{
		founderPub: founderPub,
		crl:        crl,
	}
}

// ValidatorOption يرجع خيار validator لـ kad-dht
func (v *DHTValidators) ValidatorOption() dht.Option {
	return dht.NamespacedValidator("mskt", v)
}

// Validate يتحقق من قيمة DHT
func (v *DHTValidators) Validate(key string, value []byte) error {
	switch {
	case strings.HasPrefix(key, "/mskt/identity/"):
		return v.validateIdentity(value)
	case strings.HasPrefix(key, "/mskt/domain/"):
		return v.validateDomain(value)
	case strings.HasPrefix(key, "/mskt/domain-commit/"):
		return v.validateDomainCommit(value)
	case strings.HasPrefix(key, "/mskt/revoke/"):
		return v.validateRevocation(value)
	case strings.HasPrefix(key, "/mskt/delegation/"):
		return v.validateDelegation(value)
	case strings.HasPrefix(key, "/mskt/search/"):
		return v.validateSearch(value)
	case strings.HasPrefix(key, "/mskt/prov/"):
		return nil // provider records — تحقق خفيف
	default:
		return fmt.Errorf("مفتاح DHT غير معروف: %s", key)
	}
}

// Select يختار أفضل قيمة (أعلى Sequence/Version)
func (v *DHTValidators) Select(key string, values [][]byte) (int, error) {
	switch {
	case strings.HasPrefix(key, "/mskt/identity/"):
		return selectHighestSequence(values)
	case strings.HasPrefix(key, "/mskt/domain/"):
		return selectHighestDomainVersion(values)
	default:
		if len(values) == 0 {
			return -1, fmt.Errorf("لا توجد قيم")
		}
		return 0, nil
	}
}

func (v *DHTValidators) validateIdentity(data []byte) error {
	var rec identity.IdentityRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return err
	}
	if v.crl != nil && v.crl.IsRevoked(rec.DID) {
		return fmt.Errorf("الهوية ملغاة")
	}
	return rec.Verify()
}

func (v *DHTValidators) validateDomainCommit(data []byte) error {
	_, err := naming.UnmarshalCommitRecord(data)
	return err
}

func (v *DHTValidators) validateDomain(data []byte) error {
	var rec naming.DomainRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return err
	}
	if _, err := naming.NormalizeDomainName(rec.Name); err != nil {
		return err
	}
	if rec.FounderSig == "" {
		return fmt.Errorf("توقيع المؤسس مطلوب")
	}
	// على مستوى DHT نتحقق من توقيع المؤسس؛ OwnerSig يُتحقق عند الحل الكامل
	return rec.VerifyFounderSig(v.founderPub)
}

func (v *DHTValidators) validateRevocation(data []byte) error {
	var rec identity.RevocationRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return err
	}
	pub, err := crypto.PubKeyFromHex(rec.PublicKeyHex)
	if err != nil {
		return fmt.Errorf("مفتاح عام غير صالح: %w", err)
	}
	return rec.Verify(pub)
}

func (v *DHTValidators) validateDelegation(data []byte) error {
	var rec identity.DelegationRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return err
	}
	pub, err := crypto.PubKeyFromHex(rec.OwnerPublicKeyHex)
	if err != nil {
		return fmt.Errorf("مفتاح عام غير صالح: %w", err)
	}
	return rec.Verify(pub)
}

func (v *DHTValidators) validateSearch(data []byte) error {
	var entry search.IndexEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return err
	}
	pub, err := crypto.PubKeyFromHex(entry.PublicKeyHex)
	if err != nil {
		return fmt.Errorf("مفتاح عام غير صالح: %w", err)
	}
	if err := entry.Verify(pub); err != nil {
		return err
	}
	_, err = peer.Decode(entry.PeerID)
	return err
}

func selectHighestSequence(values [][]byte) (int, error) {
	bestIdx := -1
	var bestSeq uint64
	for i, val := range values {
		var rec identity.IdentityRecord
		if err := json.Unmarshal(val, &rec); err != nil {
			continue
		}
		if bestIdx < 0 || rec.Sequence > bestSeq {
			bestIdx = i
			bestSeq = rec.Sequence
		}
	}
	if bestIdx < 0 {
		return -1, fmt.Errorf("لا توجد سجلات صالحة")
	}
	return bestIdx, nil
}

func selectHighestDomainVersion(values [][]byte) (int, error) {
	bestIdx := -1
	var bestVer uint64
	var bestOwner string
	for i, val := range values {
		var rec naming.DomainRecord
		if err := json.Unmarshal(val, &rec); err != nil {
			continue
		}
		// أعلى Version مع نفس Owner
		if bestIdx < 0 {
			bestIdx = i
			bestVer = rec.Version
			bestOwner = rec.Owner
			continue
		}
		if rec.Owner != bestOwner {
			continue // رفض تغيير Owner بدون نقل ملكية موثّق
		}
		if rec.Version > bestVer {
			bestIdx = i
			bestVer = rec.Version
		}
	}
	if bestIdx < 0 {
		return -1, fmt.Errorf("لا توجد سجلات صالحة")
	}
	return bestIdx, nil
}
