package protocol

// بروتوكولات Musketeers
const (
	ProtocolBitswap    = "/nr/bitswap/1.0.0"
	ProtocolDirect     = "/nr/direct/1.0.0"
	ProtocolVersion    = "Musketeers/1.0.0"
)

// أقصى أحجام
const (
	MaxMessageSize   = 1 << 20 // 1MB
	MaxChunkSize     = 256 << 10 // 256KB
	MaxBlockSize     = 4 << 20 // 4MB
	MaxManifestFiles = 1000
	MaxManifestSize  = 1 << 20 // 1MB
	MaxMetaSize      = 4096
)

// ChannelMessage رسالة قناة عامة
type ChannelMessage struct {
	From      string `json:"from"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

// EncryptedMessage رسالة قناة خاصة مشفرة
type EncryptedMessage struct {
	Nonce      string `json:"nonce"`      // hex
	Ciphertext string `json:"ciphertext"` // hex — يحتوي From+Content داخلياً
}

// PrivatePlaintext محتوى الرسالة الخاصة قبل التشفير
type PrivatePlaintext struct {
	From      string `json:"from"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// DirectMessage رسالة مباشرة 1:1
type DirectMessage struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Ephemeral  string `json:"ephemeral"`  // hex — مفتاح مؤقت Curve25519
	Nonce      string `json:"nonce"`      // hex
	Ciphertext string `json:"ciphertext"` // hex — NaCl box
	Timestamp  int64  `json:"timestamp"`
	Signature  string `json:"signature"`
	ChunkIndex int    `json:"chunk_index,omitempty"`
	ChunkTotal int    `json:"chunk_total,omitempty"`
	FileID     string `json:"file_id,omitempty"`
	FileHash   string `json:"file_hash,omitempty"` // sha256 للملف الكامل
}

// SiteManifest وصف موقع .ia
type SiteManifest struct {
	Version int               `json:"version"`
	Title   string            `json:"title"`
	Files   map[string]string `json:"files"` // path -> CID
}

// ProviderRecord قائمة موفري كتلة
type ProviderRecord struct {
	CID       string   `json:"cid"`
	Providers []string `json:"providers"`
}
