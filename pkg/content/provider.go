package content

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
)

const providerDHTPrefix = "/nr/prov/"

// ProviderManager يدير تسجيل الموفرين في DHT
type ProviderManager struct {
	host  host.Host
	dht   *dht.IpfsDHT
	store BlockStore
	log   *logrus.Entry
}

// NewProviderManager ينشئ مدير موفرين
func NewProviderManager(h host.Host, kad *dht.IpfsDHT, store BlockStore, log *logrus.Logger) *ProviderManager {
	return &ProviderManager{
		host:  h,
		dht:   kad,
		store: store,
		log:   log.WithField("component", "provider"),
	}
}

// PublishContent يخزّن كتلة ويسجّل نفسه كموفر
func (pm *ProviderManager) PublishContent(ctx context.Context, data []byte, did string) (string, error) {
	cid := CIDFromData(data)
	if err := pm.store.Put(cid, data, did); err != nil {
		return "", fmt.Errorf("فشل تخزين الكتلة: %w", err)
	}
	if err := pm.AddProvider(ctx, cid); err != nil {
		// العقدة المعزولة قد لا تجد peers في جدول التوجيه — التخزين المحلي كافٍ
		pm.log.WithError(err).Warn("تسجيل الموفر على DHT فشل — المحتوى مخزّن محلياً")
	} else {
		pm.log.WithField("cid", cid).Info("تم نشر المحتوى")
	}
	return cid, nil
}

// AddProvider يسجّل العقدة الحالية كموفّر لـ CID
func (pm *ProviderManager) AddProvider(ctx context.Context, cid string) error {
	key := providerDHTPrefix + cid
	rec := protocol.ProviderRecord{
		CID:       cid,
		Providers: []string{pm.host.ID().String()},
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return pm.dht.PutValue(ctx, key, data)
}

// FindProviders يبحث عن موفري CID في DHT
func (pm *ProviderManager) FindProviders(ctx context.Context, cid string) ([]peer.ID, error) {
	key := providerDHTPrefix + cid
	val, err := pm.dht.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	var rec protocol.ProviderRecord
	if err := json.Unmarshal(val, &rec); err != nil {
		return nil, err
	}
	var peers []peer.ID
	for _, pstr := range rec.Providers {
		pid, err := peer.Decode(pstr)
		if err != nil {
			continue
		}
		peers = append(peers, pid)
	}
	return peers, nil
}

// ServeBitswap يستقبل طلبات Bitswap
func (pm *ProviderManager) ServeBitswap(s network.Stream) {
	defer s.Close()
	buf := make([]byte, 128)
	n, err := s.Read(buf)
	if err != nil || n == 0 {
		return
	}
	// الطلب: CID\n
	cid := string(buf[:n])
	if len(cid) > 0 && cid[len(cid)-1] == '\n' {
		cid = cid[:len(cid)-1]
	}

	data, err := pm.store.Get(cid)
	if err != nil {
		pm.log.WithField("cid", cid).Debug("كتلة غير موجودة محلياً")
		return
	}

	// الاستجابة: 4 bytes length (big endian) + data
	length := make([]byte, 4)
	length[0] = byte(len(data) >> 24)
	length[1] = byte(len(data) >> 16)
	length[2] = byte(len(data) >> 8)
	length[3] = byte(len(data))
	if _, err := s.Write(length); err != nil {
		return
	}
	if _, err := s.Write(data); err != nil {
		return
	}
}

// RequestBlock يطلب كتلة من نظير عبر Bitswap
func RequestBlock(ctx context.Context, h host.Host, pid peer.ID, cid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	s, err := h.NewStream(ctx, pid, protocol.ProtocolBitswap)
	if err != nil {
		return nil, fmt.Errorf("فشل فتح stream: %w", err)
	}
	defer s.Close()

	req := cid + "\n"
	if _, err := s.Write([]byte(req)); err != nil {
		return nil, err
	}

	lengthBuf := make([]byte, 4)
	if _, err := s.Read(lengthBuf); err != nil {
		return nil, fmt.Errorf("فشل قراءة الطول: %w", err)
	}
	length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])
	if length > protocol.MaxBlockSize || length <= 0 {
		return nil, fmt.Errorf("حجم كتلة غير صالح: %d", length)
	}

	data := make([]byte, length)
	read := 0
	for read < length {
		n, err := s.Read(data[read:])
		if err != nil {
			return nil, err
		}
		read += n
	}

	if err := VerifyCID(cid, data); err != nil {
		return nil, fmt.Errorf("تحقق CID فشل: %w", err)
	}
	return data, nil
}
