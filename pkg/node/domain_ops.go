package node

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/naming"
	"github.com/sirupsen/logrus"
)

// PutDomainCommit ينشر سجل تزام جاهز على DHT
func (n *Node) PutDomainCommit(ctx context.Context, commit *naming.DomainCommitRecord) error {
	data, err := commit.Marshal()
	if err != nil {
		return err
	}
	return n.dht().PutValue(ctx, naming.DHTCommitKey(commit.Commitment), data)
}

// PublishDomainCommit ينشر التزام تسجيل نطاق (الاسم مخفي)
func (n *Node) PublishDomainCommit(ctx context.Context, name, owner, secret string) (*naming.DomainCommitRecord, error) {
	if !n.rateLimiter().Allow(n.host().ID().String()) {
		return nil, fmt.Errorf("تجاوز حد المعدل")
	}
	commit, err := naming.NewDomainCommitRecord(name, owner, secret)
	if err != nil {
		return nil, err
	}
	if commit.Owner != n.keyPair().DID {
		return nil, fmt.Errorf("يمكن للمالك فقط نشر التزام لنفسه")
	}
	data, err := commit.Marshal()
	if err != nil {
		return nil, err
	}
	key := naming.DHTCommitKey(commit.Commitment)
	if err := n.dht().PutValue(ctx, key, data); err != nil {
		return nil, err
	}
	n.log.WithField("commitment", commit.Commitment).Info("تم نشر التزام النطاق")
	return commit, nil
}

// GetDomainCommit يجلب سجل التزام
func (n *Node) GetDomainCommit(ctx context.Context, commitment string) (*naming.DomainCommitRecord, error) {
	val, err := n.dht().GetValue(ctx, naming.DHTCommitKey(commitment))
	if err != nil {
		return nil, fmt.Errorf("التزام غير موجود: %w", err)
	}
	return naming.UnmarshalCommitRecord(val)
}

// RegisterDomainReveal يسجّل نطاقاً بعد التحقق من commit-reveal (للمؤسس)
func (n *Node) RegisterDomainReveal(ctx context.Context, name, owner, secret, target, recordType string, expiresAt int64, founderPriv ed25519.PrivateKey) (*naming.DomainRecord, error) {
	hash, err := naming.CommitmentHash(name, owner, secret)
	if err != nil {
		return nil, err
	}
	commit, err := n.GetDomainCommit(ctx, hash)
	if err != nil {
		return nil, err
	}
	if err := naming.VerifyReveal(commit, name, owner, secret); err != nil {
		return nil, err
	}

	rec, err := naming.NewDomainRecord(name, owner, target, recordType, expiresAt, founderPriv)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}
	if err := n.dht().PutValue(ctx, rec.DHTKey(), data); err != nil {
		return nil, err
	}
	n.log.WithFields(logrus.Fields{
		"domain": rec.Name,
		"owner":  rec.Owner,
	}).Info("تم تسجيل النطاق عبر commit-reveal")
	return rec, nil
}
