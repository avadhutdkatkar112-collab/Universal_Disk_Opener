package vault

import (
	"encoding/json"
	"os"
	"time"
)

type Manifest struct {
	Magic           string       `json:"magic"`
	FormatVersion   int          `json:"format_version"`
	VaultID         string       `json:"vault_id"`
	CreatedAt       time.Time    `json:"created_at"`
	KDFParams       Argon2Params `json:"kdf_params"`
	Salt            []byte       `json:"salt"`
	Cipher          string       `json:"cipher"`
	ChunkSize       int          `json:"chunk_size"`
	WrappedDEK      []byte       `json:"wrapped_dek"`
	WrappedDEKNonce []byte       `json:"wrapped_dek_nonce"`
	TotalSize       int64        `json:"total_size_bytes"`
	TotalChunks     int64        `json:"total_chunks"`
	SourceSHA256    string       `json:"source_sha256"`
}

func NewManifest(vaultID string, salt, wrappedDEK, dekNonce []byte, sourceSHA256 string, totalSize int64, chunkSize int) *Manifest {
	totalChunks := (totalSize + int64(chunkSize) - 1) / int64(chunkSize)
	return &Manifest{
		Magic:           "DVE1",
		FormatVersion:   1,
		VaultID:         vaultID,
		CreatedAt:       time.Now().UTC(),
		KDFParams:       DefaultArgon2Params(),
		Salt:            salt,
		Cipher:          "AES-256-GCM",
		ChunkSize:       chunkSize,
		WrappedDEK:      wrappedDEK,
		WrappedDEKNonce: dekNonce,
		TotalSize:       totalSize,
		TotalChunks:     totalChunks,
		SourceSHA256:    sourceSHA256,
	}
}

func SaveManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
