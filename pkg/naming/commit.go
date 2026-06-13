package naming

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// MinRevealDelay أقل فترة انتظار بين commit و reveal
const MinRevealDelay = 24 * time.Hour

// DomainCommitRecord سجل التزام بتسجيل نطاق (الاسم مخفي)
type DomainCommitRecord struct {
	Commitment  string `json:"commitment"` // hex(sha256)
	Owner       string `json:"owner"`
	CommittedAt int64  `json:"committed_at"`
}

// CommitmentHash يحسب hash(name|owner|secret)
func CommitmentHash(name, owner, secret string) (string, error) {
	normalized, err := NormalizeDomainName(name)
	if err != nil {
		return "", err
	}
	if owner == "" || secret == "" {
		return "", fmt.Errorf("المالك والسر مطلوبان")
	}
	payload := normalized + "|" + owner + "|" + secret
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:]), nil
}

// GenerateSecret يولّد سر عشوائي للـ commit-reveal
func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// NewDomainCommitRecord ينشئ سجل التزام
func NewDomainCommitRecord(name, owner, secret string) (*DomainCommitRecord, error) {
	hash, err := CommitmentHash(name, owner, secret)
	if err != nil {
		return nil, err
	}
	return &DomainCommitRecord{
		Commitment:  hash,
		Owner:       owner,
		CommittedAt: time.Now().Unix(),
	}, nil
}

// CommitRecordWithTime ينشئ سجل تزام بوقت محدد (للاختبارات)
func CommitRecordWithTime(name, owner, secret string, committedAt int64) (*DomainCommitRecord, error) {
	hash, err := CommitmentHash(name, owner, secret)
	if err != nil {
		return nil, err
	}
	return &DomainCommitRecord{
		Commitment:  hash,
		Owner:       owner,
		CommittedAt: committedAt,
	}, nil
}

// VerifyReveal يتحقق أن الكشف يطابق التزاماً سابقاً
func VerifyReveal(commit *DomainCommitRecord, name, owner, secret string) error {
	if commit == nil {
		return fmt.Errorf("سجل التزام غير موجود")
	}
	if commit.Owner != owner {
		return fmt.Errorf("المالك لا يطابق التزام")
	}
	hash, err := CommitmentHash(name, owner, secret)
	if err != nil {
		return err
	}
	if hash != commit.Commitment {
		return fmt.Errorf("السر أو اسم النطاق لا يطابق التزام")
	}
	elapsed := time.Now().Unix() - commit.CommittedAt
	if elapsed < int64(MinRevealDelay.Seconds()) {
		return fmt.Errorf("فترة الانتظار لم تنتهِ بعد (%ds متبقية)", int64(MinRevealDelay.Seconds())-elapsed)
	}
	return nil
}

// DHTCommitKey مفتاح DHT للتزام
func DHTCommitKey(commitment string) string {
	return "/mskt/domain-commit/" + commitment
}

// Marshal يسلسل السجل
func (c *DomainCommitRecord) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// UnmarshalCommitRecord يفك سجل التزام
func UnmarshalCommitRecord(data []byte) (*DomainCommitRecord, error) {
	var rec DomainCommitRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	if rec.Commitment == "" || rec.Owner == "" {
		return nil, fmt.Errorf("سجل تزام غير صالح")
	}
	return &rec, nil
}
