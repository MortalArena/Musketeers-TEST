package channel

import (
	"crypto/ed25519"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/MortalArena/Musketeers/pkg/common"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// ChannelMessageValidator يتحقق من رسائل القنوات العامة
type ChannelMessageValidator struct {
	keyResolver common.KeyResolver
	msgFilter   func(did string) bool
	log         *logrus.Entry
}

// NewChannelMessageValidator ينشئ validator
func NewChannelMessageValidator(resolver common.KeyResolver, log *logrus.Logger) *ChannelMessageValidator {
	return &ChannelMessageValidator{
		keyResolver: resolver,
		log:         log.WithField("component", "channel-validator"),
	}
}

// SetFilter يضبط فلتر الرسائل
func (v *ChannelMessageValidator) SetFilter(f func(did string) bool) {
	v.msgFilter = f
}

// ChannelMsgPayload payload توقيع رسالة القناة
func ChannelMsgPayload(channelID string, msg *protocol.ChannelMessage) string {
	return strings.Join([]string{
		channelID,
		msg.From,
		msg.Content,
		strconv.FormatInt(msg.Timestamp, 10),
	}, "|")
}

// SignChannelMessage يوقّع رسالة قناة
func SignChannelMessage(channelID string, msg *protocol.ChannelMessage, priv ed25519.PrivateKey) error {
	payload := ChannelMsgPayload(channelID, msg)
	domain := nrcrypto.DomainChannelMsg + channelID + "|"
	sig, err := nrcrypto.SignPayloadHex(priv, domain, payload)
	if err != nil {
		return err
	}
	msg.Signature = sig
	return nil
}

// Validate يتحقق من رسالة واردة — يُستخدم كـ pubsub validator
func (v *ChannelMessageValidator) Validate(channelID string, data []byte) pubsub.ValidationResult {
	// فحص الحجم قبل فك التوقيع (حماية DoS)
	if len(data) > protocol.MaxMessageSize {
		v.log.Debug("رسالة تتجاوز الحد الأقصى")
		return pubsub.ValidationReject
	}

	var msg protocol.ChannelMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return pubsub.ValidationReject
	}

	if msg.Signature == "" {
		v.log.Debug("رسالة بدون توقيع")
		return pubsub.ValidationReject
	}

	if v.msgFilter != nil && v.msgFilter(msg.From) {
		return pubsub.ValidationReject
	}

	pub, err := v.keyResolver.ResolvePublicKey(msg.From)
	if err != nil {
		v.log.WithError(err).Debug("فشل حل المفتاح")
		return pubsub.ValidationReject
	}

	payload := ChannelMsgPayload(channelID, &msg)
	domain := nrcrypto.DomainChannelMsg + channelID + "|"
	if err := nrcrypto.VerifyPayloadHex(pub, domain, payload, msg.Signature); err != nil {
		v.log.WithError(err).Debug("توقيع غير صالح")
		return pubsub.ValidationReject
	}

	return pubsub.ValidationAccept
}

// NewChannelMessage ينشئ رسالة قناة موقّعة
func NewChannelMessage(from, content string, channelID string, priv ed25519.PrivateKey) (*protocol.ChannelMessage, error) {
	msg := &protocol.ChannelMessage{
		From:      from,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
	if err := SignChannelMessage(channelID, msg, priv); err != nil {
		return nil, err
	}
	return msg, nil
}

// TopicName يبني اسم topic للقناة
func TopicName(channelID string) string {
	return "nr/channel/" + channelID
}
