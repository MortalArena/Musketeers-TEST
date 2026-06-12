package identity

import (
	"crypto/ed25519"
	"fmt"
	"strconv"
	"strings"
	"time"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
)

// صلاحيات التفويض المعروفة
const (
	PermSendMessage  = "send_message"
	PermUpdateDomain = "update_domain"
	PermPublishSearch = "publish_search"
)

// DelegationRecord سجل تفويض الصلاحيات
type DelegationRecord struct {
	Owner             string   `json:"owner"`
	Delegate          string   `json:"delegate"`
	Permissions       []string `json:"permissions"`
	ExpiresAt         int64    `json:"expires_at"`
	OwnerPublicKeyHex string   `json:"owner_public_key_hex,omitempty"`
	Signature         string   `json:"signature"`
}

// DelegationPayload payload ثابت للتوقيع
func DelegationPayload(rec *DelegationRecord) string {
	perms := strings.Join(rec.Permissions, ",")
	return strings.Join([]string{
		rec.Owner,
		rec.Delegate,
		perms,
		strconv.FormatInt(rec.ExpiresAt, 10),
	}, "|")
}

// NewDelegationRecord ينشئ سجل تفويض
func NewDelegationRecord(owner, delegate string, permissions []string, expiresAt int64, priv ed25519.PrivateKey) (*DelegationRecord, error) {
	rec := &DelegationRecord{
		Owner:             owner,
		Delegate:          delegate,
		Permissions:       permissions,
		ExpiresAt:         expiresAt,
		OwnerPublicKeyHex: nrcrypto.PublicKeyHex(priv.Public().(ed25519.PublicKey)),
	}
	payload := DelegationPayload(rec)
	sig, err := nrcrypto.SignPayloadHex(priv, nrcrypto.DomainDelegation, payload)
	if err != nil {
		return nil, err
	}
	rec.Signature = sig
	return rec, nil
}

// Verify يتحقق من سجل التفويض
func (rec *DelegationRecord) Verify(ownerPub ed25519.PublicKey) error {
	if rec.Owner == "" || rec.Delegate == "" || rec.Signature == "" {
		return fmt.Errorf("حقول مطلوبة ناقصة")
	}
	if nrcrypto.DIDFromPublicKey(ownerPub) != rec.Owner {
		return fmt.Errorf("DID لا يطابق المفتاح العام")
	}
	if time.Now().Unix() > rec.ExpiresAt {
		return fmt.Errorf("التفويض منتهي الصلاحية")
	}
	payload := DelegationPayload(rec)
	return nrcrypto.VerifyPayloadHex(ownerPub, nrcrypto.DomainDelegation, payload, rec.Signature)
}

// HasPermission يتحقق من وجود صلاحية معينة
func (rec *DelegationRecord) HasPermission(perm string) bool {
	for _, p := range rec.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// DHTKey مفتاح DHT للتفويض
func (rec *DelegationRecord) DHTKey() string {
	return "/nr/delegation/" + rec.Owner + "/" + rec.Delegate
}

// ValidatePermissionsSubset يتحقق أن الصلاحيات المفوّضة subset من صلاحيات المالك
func ValidatePermissionsSubset(granted, ownerCaps []string) bool {
	ownerSet := make(map[string]bool, len(ownerCaps))
	for _, c := range ownerCaps {
		ownerSet[c] = true
	}
	for _, g := range granted {
		if !ownerSet[g] {
			return false
		}
	}
	return true
}
