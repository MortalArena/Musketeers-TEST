package node

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"filippo.io/edwards25519"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/protocol"
	"github.com/dgraph-io/badger/v4"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

// NonceStore يخزّن nonces لمنع replay
type NonceStore struct {
	db     *badger.DB
	prefix []byte
	ttl    time.Duration
}

// NewNonceStore ينشئ مخزن nonces
func NewNonceStore(db *badger.DB, ttl time.Duration) *NonceStore {
	return &NonceStore{
		db:     db,
		prefix: []byte("nonce:"),
		ttl:    ttl,
	}
}

func (ns *NonceStore) key(nonce string) []byte {
	k := make([]byte, len(ns.prefix)+len(nonce))
	copy(k, ns.prefix)
	copy(k[len(ns.prefix):], nonce)
	return k
}

// Seen يتحقق إن كان nonce مستخدم — يُسجّله إن لم يكن
func (ns *NonceStore) Seen(nonce string) (bool, error) {
	seen := false
	err := ns.db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(ns.key(nonce))
		if err == nil {
			seen = true
			return nil
		}
		if err != badger.ErrKeyNotFound {
			return err
		}
		entry := badger.NewEntry(ns.key(nonce), []byte{1}).WithTTL(ns.ttl)
		return txn.SetEntry(entry)
	})
	return seen, err
}

// Cleanup يزيل nonces منتهية (Badger يتولى TTL تلقائياً)
func (ns *NonceStore) Cleanup() {
	// Badger TTL يتولى التنظيف
}

// DirectMsgPayload payload توقيع الرسالة المباشرة
func DirectMsgPayload(msg *protocol.DirectMessage) string {
	return strings.Join([]string{
		msg.From,
		msg.To,
		msg.Ephemeral,
		msg.Nonce,
		msg.Ciphertext,
		strconv.FormatInt(msg.Timestamp, 10),
	}, "|")
}

// EncryptDirectMessage يشفّر رسالة مباشرة بـ NaCl box
func EncryptDirectMessage(from, to string, content []byte, senderPriv ed25519.PrivateKey, recipientPub ed25519.PublicKey) (*protocol.DirectMessage, error) {
	// مفتاح مؤقت Curve25519
	var ephemeralPub, ephemeralPriv [32]byte
	if _, err := rand.Read(ephemeralPriv[:]); err != nil {
		return nil, err
	}
	curve25519.ScalarBaseMult(&ephemeralPub, &ephemeralPriv)

	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	recipientCurve := convertToCurve25519(recipientPub)
	encrypted := box.Seal(nil, content, (*[24]byte)(nonce), &recipientCurve, &ephemeralPriv)

	msg := &protocol.DirectMessage{
		From:       from,
		To:         to,
		Ephemeral:  hex.EncodeToString(ephemeralPub[:]),
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(encrypted),
		Timestamp:  time.Now().Unix(),
	}

	payload := DirectMsgPayload(msg)
	sig, err := nrcrypto.SignPayloadHex(senderPriv, nrcrypto.DomainDirectMsg, payload)
	if err != nil {
		return nil, err
	}
	msg.Signature = sig
	return msg, nil
}

// DecryptDirectMessage يفك تشفير رسالة مباشرة
func DecryptDirectMessage(msg *protocol.DirectMessage, recipientPriv ed25519.PrivateKey, senderPub ed25519.PublicKey) ([]byte, error) {
	ephemeralPub, err := hex.DecodeString(msg.Ephemeral)
	if err != nil || len(ephemeralPub) != 32 {
		return nil, fmt.Errorf("مفتاح مؤقت غير صالح")
	}
	nonce, err := hex.DecodeString(msg.Nonce)
	if err != nil || len(nonce) != 24 {
		return nil, fmt.Errorf("nonce غير صالح")
	}
	ciphertext, err := hex.DecodeString(msg.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("ciphertext غير صالح")
	}

	// التحقق من التوقيع
	payload := DirectMsgPayload(msg)
	if err := nrcrypto.VerifyPayloadHex(senderPub, nrcrypto.DomainDirectMsg, payload, msg.Signature); err != nil {
		return nil, fmt.Errorf("توقيع غير صالح: %w", err)
	}

	var ephemeralPubArr [32]byte
	copy(ephemeralPubArr[:], ephemeralPub)
	recipientCurve := convertToCurve25519FromPriv(recipientPriv)

	plain, ok := box.Open(nil, ciphertext, (*[24]byte)(nonce), &ephemeralPubArr, &recipientCurve)
	if !ok {
		return nil, fmt.Errorf("فشل فك التشفير")
	}
	return plain, nil
}

// convertToCurve25519 يحوّل Ed25519 public key إلى Curve25519
func convertToCurve25519(pub ed25519.PublicKey) [32]byte {
	var curve [32]byte
	p, err := new(edwards25519.Point).SetBytes(pub)
	if err != nil {
		return curve
	}
	copy(curve[:], p.BytesMontgomery())
	return curve
}

// convertToCurve25519FromPriv يحوّل Ed25519 private key seed إلى Curve25519
func convertToCurve25519FromPriv(priv ed25519.PrivateKey) [32]byte {
	var curve [32]byte
	seed := priv.Seed()
	h := sha512.Sum512(seed)
	copy(curve[:], h[:32])
	curve[0] &= 248
	curve[31] &= 127
	curve[31] |= 64
	return curve
}

// ChunkMessage يقسّم محتوى كبير إلى رسائل مجزأة
func ChunkMessage(from, to string, content []byte, senderPriv ed25519.PrivateKey, recipientPub ed25519.PublicKey) ([]*protocol.DirectMessage, error) {
	if len(content) <= protocol.MaxChunkSize {
		msg, err := EncryptDirectMessage(from, to, content, senderPriv, recipientPub)
		if err != nil {
			return nil, err
		}
		return []*protocol.DirectMessage{msg}, nil
	}

	sum := sha256.Sum256(content)
	fileHash := hex.EncodeToString(sum[:])
	fileIDNonce := make([]byte, 8)
	if _, err := rand.Read(fileIDNonce); err != nil {
		return nil, err
	}
	fileID := hex.EncodeToString(fileIDNonce)

	total := (len(content) + protocol.MaxChunkSize - 1) / protocol.MaxChunkSize
	var msgs []*protocol.DirectMessage

	for i := 0; i < total; i++ {
		start := i * protocol.MaxChunkSize
		end := start + protocol.MaxChunkSize
		if end > len(content) {
			end = len(content)
		}
		chunk := content[start:end]

		msg, err := EncryptDirectMessage(from, to, chunk, senderPriv, recipientPub)
		if err != nil {
			return nil, err
		}
		msg.ChunkIndex = i
		msg.ChunkTotal = total
		msg.FileID = fileID
		if i == 0 {
			msg.FileHash = fileHash
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// ChunkAssembler يجمع الأجزاء بشكل متزامن مع حماية من تسريبات الذاكرة
type ChunkAssembler struct {
	mu         sync.RWMutex
	chunks     map[string]map[int][]byte
	totals     map[string]int
	timestamps map[string]time.Time
	stopCh     chan struct{}
}

// NewChunkAssembler ينشئ مجمّعاً مع تشغيل حلقة تنظيف خلفية
func NewChunkAssembler() *ChunkAssembler {
	ca := &ChunkAssembler{
		chunks:     make(map[string]map[int][]byte),
		totals:     make(map[string]int),
		timestamps: make(map[string]time.Time),
		stopCh:     make(chan struct{}),
	}
	// حلقة تنظيف دورية كل 5 دقائق لمسح الملفات غير المكتملة التي تجاوز عمرها 30 دقيقة
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ca.Cleanup(30 * time.Minute)
			case <-ca.stopCh:
				return
			}
		}
	}()
	return ca
}

// Cleanup يزيل الأجزاء القديمة من الذاكرة
func (ca *ChunkAssembler) Cleanup(maxAge time.Duration) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for id, ts := range ca.timestamps {
		if ts.Before(cutoff) {
			delete(ca.chunks, id)
			delete(ca.totals, id)
			delete(ca.timestamps, id)
		}
	}
}

func (ca *ChunkAssembler) Close() error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	select {
	case <-ca.stopCh:
		// Already closed
		return nil
	default:
		close(ca.stopCh)
		return nil
	}
}

// Add يضيف جزءاً ويتحقق من بصمة الملف الكامل عند الاكتمال
func (ca *ChunkAssembler) Add(msg *protocol.DirectMessage, data []byte) (complete []byte, done bool, err error) {
	if msg.FileID == "" || msg.ChunkTotal <= 1 {
		return data, true, nil
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()

	if ca.chunks[msg.FileID] == nil {
		ca.chunks[msg.FileID] = make(map[int][]byte)
	}
	ca.chunks[msg.FileID][msg.ChunkIndex] = data
	ca.totals[msg.FileID] = msg.ChunkTotal
	ca.timestamps[msg.FileID] = time.Now()

	if len(ca.chunks[msg.FileID]) < msg.ChunkTotal {
		return nil, false, nil
	}

	var result []byte
	for i := 0; i < msg.ChunkTotal; i++ {
		chunk, ok := ca.chunks[msg.FileID][i]
		if !ok {
			return nil, false, nil
		}
		result = append(result, chunk...)
	}

	delete(ca.chunks, msg.FileID)
	delete(ca.totals, msg.FileID)
	delete(ca.timestamps, msg.FileID)

	// التحقق من بصمة الملف الكامل
	if msg.FileHash != "" {
		sum := sha256.Sum256(result)
		hashStr := hex.EncodeToString(sum[:])
		if hashStr != msg.FileHash {
			return nil, true, fmt.Errorf("بصمة الملف المجزأ لا تطابق القيمة المتوقعة (integrity check failed)")
		}
	}

	return result, true, nil
}

// SendDirect يرسل رسالة مباشرة عبر stream
func SendDirect(w io.Writer, msg *protocol.DirectMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if len(data) > protocol.MaxMessageSize {
		return fmt.Errorf("الرسالة كبيرة جداً")
	}
	_, err = w.Write(data)
	return err
}

// ReadDirect يقرأ رسالة مباشرة
func ReadDirect(r io.Reader) (*protocol.DirectMessage, error) {
	data, err := io.ReadAll(io.LimitReader(r, protocol.MaxMessageSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > protocol.MaxMessageSize {
		return nil, fmt.Errorf("الرسالة تتجاوز الحد الأقصى")
	}
	var msg protocol.DirectMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
