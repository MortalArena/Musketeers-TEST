package channel

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
)

// ChannelConfigPayloadV2 payload موقّع يشمل إصدار المفتاح
func ChannelConfigPayloadV2(cfg *ChannelConfig) string {
	members := strings.Join(cfg.Members, ",")
	admins := strings.Join(cfg.Admins, ",")
	memberKeys := ""
	if len(cfg.MemberKeys) > 0 {
		parts := make([]string, 0, len(cfg.MemberKeys))
		for did, enc := range cfg.MemberKeys {
			parts = append(parts, did+":"+enc)
		}
		// ترتيب ثابت
		for i := 0; i < len(parts); i++ {
			for j := i + 1; j < len(parts); j++ {
				if parts[i] > parts[j] {
					parts[i], parts[j] = parts[j], parts[i]
				}
			}
		}
		memberKeys = strings.Join(parts, ";")
	}
	return strings.Join([]string{
		cfg.ID,
		cfg.Owner,
		members,
		admins,
		cfg.SharedKey,
		strconv.FormatUint(cfg.KeyVersion, 10),
		memberKeys,
	}, "|")
}

// signConfig يوقّع الإعدادات
func signConfig(cfg *ChannelConfig, ownerPriv ed25519.PrivateKey) error {
	payload := ChannelConfigPayloadV2(cfg)
	sig, err := nrcrypto.SignPayloadHex(ownerPriv, nrcrypto.DomainChannelConfig, payload)
	if err != nil {
		return err
	}
	cfg.Signature = sig
	return nil
}

// EncryptKeyForMember يشفّر مفتاح AES لمفتاح عام عضو
func EncryptKeyForMember(aesKey []byte, memberPub ed25519.PublicKey) (string, error) {
	enc, err := encryptKeyECDH(aesKey, memberPub)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(enc), nil
}

// AddMember يضيف عضواً ويوزّع المفتاح المشفّر له
func (cfg *ChannelConfig) AddMember(memberDID string, memberPub ed25519.PublicKey, actorDID string, ownerPriv ed25519.PrivateKey, currentKey []byte) error {
	if !cfg.IsAdmin(actorDID) {
		return fmt.Errorf("صلاحية غير كافية")
	}
	if cfg.IsMember(memberDID) {
		return fmt.Errorf("العضو موجود مسبقاً")
	}
	if cfg.MemberKeys == nil {
		cfg.MemberKeys = make(map[string]string)
	}
	enc, err := EncryptKeyForMember(currentKey, memberPub)
	if err != nil {
		return err
	}
	cfg.Members = append(cfg.Members, memberDID)
	cfg.MemberKeys[memberDID] = enc
	return signConfig(cfg, ownerPriv)
}

// RemoveMember يزيل عضواً ويدوّر مفتاح القناة (key rotation)
func (cfg *ChannelConfig) RemoveMember(memberDID, actorDID string, ownerPriv ed25519.PrivateKey, memberPubs map[string]ed25519.PublicKey) ([]byte, error) {
	if !cfg.IsAdmin(actorDID) {
		return nil, fmt.Errorf("صلاحية غير كافية")
	}
	if memberDID == cfg.Owner {
		return nil, fmt.Errorf("لا يمكن إزالة المالك")
	}

	// إزالة العضو
	found := false
	newMembers := make([]string, 0, len(cfg.Members))
	for _, m := range cfg.Members {
		if m == memberDID {
			found = true
			continue
		}
		newMembers = append(newMembers, m)
	}
	if !found {
		return nil, fmt.Errorf("العضو غير موجود")
	}
	cfg.Members = newMembers

	// إزالة من Admins إن وُجد
	newAdmins := make([]string, 0, len(cfg.Admins))
	for _, a := range cfg.Admins {
		if a != memberDID {
			newAdmins = append(newAdmins, a)
		}
	}
	cfg.Admins = newAdmins

	// تدوير المفتاح — العضو المطرود لا يستطيع قراءة الرسائل الجديدة
	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return nil, err
	}

	ownerPub := ownerPriv.Public().(ed25519.PublicKey)
	encOwner, err := encryptKeyECDH(newKey, ownerPub)
	if err != nil {
		return nil, err
	}
	cfg.SharedKey = hex.EncodeToString(encOwner)
	cfg.KeyVersion++

	cfg.MemberKeys = make(map[string]string)
	for _, m := range cfg.Members {
		pub, ok := memberPubs[m]
		if !ok {
			continue
		}
		enc, err := EncryptKeyForMember(newKey, pub)
		if err != nil {
			return nil, err
		}
		cfg.MemberKeys[m] = enc
	}

	if err := signConfig(cfg, ownerPriv); err != nil {
		return nil, err
	}
	return newKey, nil
}

// RotateKey يدوّر المفتاح يدوياً (مثلاً بعد عدد كبير من الرسائل)
func (cfg *ChannelConfig) RotateKey(ownerPriv ed25519.PrivateKey, memberPubs map[string]ed25519.PublicKey) ([]byte, error) {
	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return nil, err
	}
	ownerPub := ownerPriv.Public().(ed25519.PublicKey)
	encOwner, err := encryptKeyECDH(newKey, ownerPub)
	if err != nil {
		return nil, err
	}
	cfg.SharedKey = hex.EncodeToString(encOwner)
	cfg.KeyVersion++
	cfg.MemberKeys = make(map[string]string)
	for _, m := range cfg.Members {
		pub, ok := memberPubs[m]
		if !ok {
			continue
		}
		enc, err := EncryptKeyForMember(newKey, pub)
		if err != nil {
			return nil, err
		}
		cfg.MemberKeys[m] = enc
	}
	if err := signConfig(cfg, ownerPriv); err != nil {
		return nil, err
	}
	return newKey, nil
}

// VerifyConfigV2 يتحقق من التوقيع مع دعم KeyVersion
func (cfg *ChannelConfig) VerifyConfigV2(ownerPub ed25519.PublicKey) error {
	payload := ChannelConfigPayloadV2(cfg)
	return nrcrypto.VerifyPayloadHex(ownerPub, nrcrypto.DomainChannelConfig, payload, cfg.Signature)
}

// MemberKeyFor يجلب المفتاح المشفّر لعضو
func (cfg *ChannelConfig) MemberKeyFor(did string) (string, bool) {
	if cfg.MemberKeys == nil {
		return "", false
	}
	enc, ok := cfg.MemberKeys[did]
	return enc, ok
}
