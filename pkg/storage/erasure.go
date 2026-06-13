package storage

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
)

const (
	DataShards   = 10
	ParityShards = 4
	TotalShards  = DataShards + ParityShards
)

// ErasureCoder يدير تجزئة وإعادة بناء البيانات
type ErasureCoder struct {
	enc reedsolomon.Encoder
}

// NewErasureCoder ينشئ مشفر تجزيئي جديد
func NewErasureCoder() (*ErasureCoder, error) {
	enc, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, fmt.Errorf("failed to create reed-solomon encoder: %w", err)
	}
	return &ErasureCoder{enc: enc}, nil
}

// Encode يقسم البيانات إلى أجزاء مشفرة
func (e *ErasureCoder) Encode(data []byte) ([][]byte, error) {
	// تقسيم البيانات
	shards, err := e.enc.Split(data)
	if err != nil {
		return nil, fmt.Errorf("failed to split data: %w", err)
	}

	// إنشاء أجزاء التكافؤ
	err = e.enc.Encode(shards)
	if err != nil {
		return nil, fmt.Errorf("failed to encode parity: %w", err)
	}

	return shards, nil
}

// Reconstruct يعيد بناء البيانات الأصلية من الأجزاء المتاحة
func (e *ErasureCoder) Reconstruct(shards [][]byte) ([]byte, error) {
	// التحقق من صحة الأجزاء وإصلاح المفقود
	err := e.enc.Reconstruct(shards)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct shards: %w", err)
	}

	// التحقق من سلامة البيانات بعد الإصلاح
	ok, err := e.enc.Verify(shards)
	if err != nil || !ok {
		return nil, fmt.Errorf("data verification failed after reconstruction")
	}

	// دمج الأجزاء للبيانات الأصلية باستخدام Join
	buf := make([]byte, 0, len(shards[0])*DataShards)
	for _, shard := range shards[:DataShards] {
		buf = append(buf, shard...)
	}

	return buf, nil
}
