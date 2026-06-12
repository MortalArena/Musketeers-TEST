package naming

import (
	"crypto/ed25519"
	"fmt"
	"strconv"
	"strings"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
)

// DomainRecord سجل النطاق على DHT
type DomainRecord struct {
	Name         string   `json:"name"`
	Owner        string   `json:"owner"`
	Target       string   `json:"target"`
	Type         string   `json:"type"` // "did", "cid", "url"
	ExpiresAt    int64    `json:"expires_at"`
	Version      uint64   `json:"version"`
	ManifestCID  string   `json:"manifest_cid,omitempty"`
	Providers    []string `json:"providers,omitempty"`
	UpdatedAt    int64    `json:"updated_at"`
	FounderSig   string   `json:"founder_sig"`
	OwnerSig     string   `json:"owner_sig,omitempty"`
}

// FounderPayload payload توقيع المؤسس
func FounderPayload(name, owner string, expiresAt int64) string {
	return strings.Join([]string{name, owner, strconv.FormatInt(expiresAt, 10)}, ":")
}

// OwnerPayload payload توقيع المالك
func OwnerPayload(name, owner, target string, version uint64) string {
	return strings.Join([]string{
		name, owner, target, strconv.FormatUint(version, 10),
	}, ":")
}

// NewDomainRecord ينشئ سجل نطاق جديد (تسجيل من المؤسس)
func NewDomainRecord(name, owner, target, recordType string, expiresAt int64, founderPriv ed25519.PrivateKey) (*DomainRecord, error) {
	normalized, err := NormalizeDomainName(name)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	rec := &DomainRecord{
		Name:      normalized,
		Owner:     owner,
		Target:    target,
		Type:      recordType,
		ExpiresAt: expiresAt,
		Version:   1,
		UpdatedAt: now,
	}

	payload := FounderPayload(rec.Name, rec.Owner, rec.ExpiresAt)
	sig, err := nrcrypto.SignPayloadHex(founderPriv, nrcrypto.DomainDomainFounder, payload)
	if err != nil {
		return nil, err
	}
	rec.FounderSig = sig
	return rec, nil
}

// SignOwner يوقّع المالك على السجل
func (rec *DomainRecord) SignOwner(ownerPriv ed25519.PrivateKey) error {
	payload := OwnerPayload(rec.Name, rec.Owner, rec.Target, rec.Version)
	sig, err := nrcrypto.SignPayloadHex(ownerPriv, nrcrypto.DomainDomainOwner, payload)
	if err != nil {
		return err
	}
	rec.OwnerSig = sig
	rec.UpdatedAt = time.Now().Unix()
	return nil
}

// UpdateByOwner يحدّث النطاق من المالك
func (rec *DomainRecord) UpdateByOwner(target, manifestCID string, providers []string, ownerPriv ed25519.PrivateKey) error {
	if len(providers) > MaxProviders {
		return fmt.Errorf("عدد الموفرين يتجاوز الحد (%d)", MaxProviders)
	}
	rec.Target = target
	rec.ManifestCID = manifestCID
	rec.Providers = providers
	rec.Version++
	return rec.SignOwner(ownerPriv)
}

// VerifyFounderSig يتحقق من توقيع المؤسس
func (rec *DomainRecord) VerifyFounderSig(founderPub ed25519.PublicKey) error {
	payload := FounderPayload(rec.Name, rec.Owner, rec.ExpiresAt)
	return nrcrypto.VerifyPayloadHex(founderPub, nrcrypto.DomainDomainFounder, payload, rec.FounderSig)
}

// VerifyOwnerSig يتحقق من توقيع المالك
func (rec *DomainRecord) VerifyOwnerSig(ownerPub ed25519.PublicKey) error {
	if rec.OwnerSig == "" {
		return nil // اختياري عند التسجيل الأول
	}
	payload := OwnerPayload(rec.Name, rec.Owner, rec.Target, rec.Version)
	return nrcrypto.VerifyPayloadHex(ownerPub, nrcrypto.DomainDomainOwner, payload, rec.OwnerSig)
}

// Verify يتحقق الكامل من السجل
func (rec *DomainRecord) Verify(founderPub, ownerPub ed25519.PublicKey) error {
	if _, err := NormalizeDomainName(rec.Name); err != nil {
		return err
	}
	if time.Now().Unix() > rec.ExpiresAt {
		return fmt.Errorf("النطاق منتهي الصلاحية")
	}
	if err := rec.VerifyFounderSig(founderPub); err != nil {
		return fmt.Errorf("توقيع المؤسس غير صالح: %w", err)
	}
	if ownerPub != nil {
		if err := rec.VerifyOwnerSig(ownerPub); err != nil {
			return fmt.Errorf("توقيع المالك غير صالح: %w", err)
		}
	}
	return nil
}

// DHTKey مفتاح DHT
func (rec *DomainRecord) DHTKey() string {
	return DHTKey(rec.Name)
}

// IsExpired يتحقق من انتهاء الصلاحية
func (rec *DomainRecord) IsExpired() bool {
	return time.Now().Unix() > rec.ExpiresAt
}
