package acp

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/MortalArena/Musketeers/pkg/common"
	"github.com/MortalArena/Musketeers/pkg/protocol"
	"github.com/sirupsen/logrus"
)

const maxACPMessageSize = protocol.MaxMessageSize

// Transport نقل ACP عبر libp2p
type Transport struct {
	host     host.Host
	router   *Router
	resolver common.KeyResolver
	fromDID  string
	priv     ed25519.PrivateKey
	log      *logrus.Entry
}

// NewTransport ينشئ ناقل ACP
func NewTransport(h host.Host, fromDID string, priv ed25519.PrivateKey, resolver common.KeyResolver, router *Router, log *logrus.Logger) *Transport {
	return &Transport{
		host:     h,
		router:   router,
		resolver: resolver,
		fromDID:  fromDID,
		priv:     priv,
		log:      log.WithField("component", "acp"),
	}
}

// ServeStream يعالج stream واردة
func (t *Transport) ServeStream(s network.Stream) {
	defer s.Close()
	env, err := readEnvelope(s)
	if err != nil {
		t.log.WithError(err).Debug("فشل قراءة ACP")
		return
	}

	senderPub, err := t.resolver.ResolvePublicKey(env.From)
	if err != nil {
		t.log.WithError(err).Debug("فشل حل مفتاح المرسل")
		return
	}
	if err := VerifyEnvelope(env, senderPub); err != nil {
		t.log.WithError(err).Debug("توقيع ACP غير صالح")
		return
	}

	if env.Intent != IntentTaskRequest {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := t.router.Handle(ctx, env.Task, env.Input)
	var resp *Envelope
	if err != nil {
		resp, _ = NewTaskError(t.fromDID, env.From, env.Task, env.RequestID, err.Error(), t.priv)
	} else {
		resp, err = NewTaskResponse(t.fromDID, env.From, env.Task, env.RequestID, output, t.priv)
		if err != nil {
			return
		}
	}
	if err := writeEnvelope(s, resp); err != nil {
		t.log.WithError(err).Debug("فشل إرسال استجابة ACP")
	}
}

// SendTask يرسل مهمة وينتظر الاستجابة
func (t *Transport) SendTask(ctx context.Context, pid peer.ID, toDID, task string, input json.RawMessage, requestID string) (*Envelope, error) {
	if requestID == "" {
		requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}

	req, err := NewTaskRequest(t.fromDID, toDID, task, input, t.fromDID, requestID, t.priv)
	if err != nil {
		return nil, err
	}

	s, err := t.host.NewStream(ctx, pid, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("فشل فتح stream ACP: %w", err)
	}
	defer s.Close()

	if err := writeEnvelope(s, req); err != nil {
		return nil, err
	}

	resp, err := readEnvelope(s)
	if err != nil {
		return nil, err
	}

	recipientPub, err := t.resolver.ResolvePublicKey(toDID)
	if err != nil {
		return nil, err
	}
	if err := VerifyEnvelope(resp, recipientPub); err != nil {
		return nil, fmt.Errorf("استجابة ACP غير موثوقة: %w", err)
	}
	return resp, nil
}

func writeEnvelope(w io.Writer, env *Envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if len(data) > maxACPMessageSize {
		return fmt.Errorf("رسالة ACP كبيرة جداً")
	}
	length := []byte{
		byte(len(data) >> 24),
		byte(len(data) >> 16),
		byte(len(data) >> 8),
		byte(len(data)),
	}
	if _, err := w.Write(length); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readEnvelope(r io.Reader) (*Envelope, error) {
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lengthBuf); err != nil {
		return nil, err
	}
	length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])
	if length <= 0 || length > maxACPMessageSize {
		return nil, fmt.Errorf("حجم رسالة ACP غير صالح")
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}
