package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const DefaultChunkSize = 4 * 1024 * 1024

func IngestEvidenceFile(srcPath, outputDir, passphrase, vaultID string) (*Manifest, error) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open evidence source: %w", err)
	}
	defer srcFile.Close()

	fi, err := srcFile.Stat()
	if err != nil {
		return nil, err
	}

	evidenceDir := filepath.Join(outputDir, "evidence")
	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		return nil, err
	}
	auditDir := filepath.Join(outputDir, "audit")
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return nil, err
	}

	salt, err := GenerateRandomBytes(16)
	if err != nil {
		return nil, err
	}

	dek, err := GenerateRandomBytes(32)
	if err != nil {
		return nil, err
	}

	kek := DeriveKEK([]byte(passphrase), salt, DefaultArgon2Params())
	wrappedDEK, dekNonce, err := WrapKey(kek, dek)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, DefaultChunkSize)
	hasher := sha256.New()
	var chunkIdx int64 = 0

	for {
		n, readErr := io.ReadFull(srcFile, buffer)
		if n > 0 {
			chunkData := buffer[:n]
			hasher.Write(chunkData)

			cipherText, nonce, err := EncryptChunk(dek, chunkData)
			if err != nil {
				return nil, fmt.Errorf("failed encrypting chunk %d: %w", chunkIdx, err)
			}

			chunkFile := filepath.Join(evidenceDir, fmt.Sprintf("%08d.chunk", chunkIdx))
			payload := append(cipherText, nonce...)
			if err := os.WriteFile(chunkFile, payload, 0600); err != nil {
				return nil, fmt.Errorf("failed writing chunk %d: %w", chunkIdx, err)
			}
			chunkIdx++
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}

	sourceSHA256 := hex.EncodeToString(hasher.Sum(nil))

	manifest := NewManifest(vaultID, salt, wrappedDEK, dekNonce, sourceSHA256, fi.Size(), DefaultChunkSize)
	if err := SaveManifest(filepath.Join(outputDir, "manifest.json"), manifest); err != nil {
		return nil, err
	}

	auditPath := filepath.Join(auditDir, "audit.log")
	logger, err := NewAuditLogger(auditPath)
	if err == nil {
		logger.LogRecord("system", "evidence.ingest", vaultID, map[string]string{
			"source":      srcPath,
			"sha256":      sourceSHA256,
			"total_bytes": fmt.Sprintf("%d", fi.Size()),
			"chunks":      fmt.Sprintf("%d", chunkIdx),
		})
	}

	return manifest, nil
}

func ExtractEvidenceChunk(caseDir string, chunkIdx int64, dek []byte) ([]byte, error) {
	chunkFile := filepath.Join(caseDir, "evidence", fmt.Sprintf("%08d.chunk", chunkIdx))
	data, err := os.ReadFile(chunkFile)
	if err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(filepath.Join(caseDir, "manifest.json"))
	if err != nil {
		return nil, err
	}

	nonceSize := 12
	ciphertext := data[:len(data)-nonceSize]
	nonce := data[len(data)-nonceSize:]

	_ = manifest
	return DecryptChunk(dek, ciphertext, nonce)
}

func ExtractEvidenceFile(caseDir, outputPath, passphrase string) error {
	manifest, err := LoadManifest(filepath.Join(caseDir, "manifest.json"))
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	salt := manifest.Salt
	kek := DeriveKEK([]byte(passphrase), salt, manifest.KDFParams)
	dek, err := UnwrapKey(kek, manifest.WrappedDEK, manifest.WrappedDEKNonce)
	if err != nil {
		return fmt.Errorf("failed to unwrap DEK: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	nonceSize := 12
	for i := int64(0); i < manifest.TotalChunks; i++ {
		chunkFile := filepath.Join(caseDir, "evidence", fmt.Sprintf("%08d.chunk", i))
		data, err := os.ReadFile(chunkFile)
		if err != nil {
			return fmt.Errorf("failed to read chunk %d: %w", i, err)
		}

		ciphertext := data[:len(data)-nonceSize]
		nonce := data[len(data)-nonceSize:]

		plaintext, err := DecryptChunk(dek, ciphertext, nonce)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk %d: %w", i, err)
		}

		if _, err := outFile.Write(plaintext); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", i, err)
		}
	}

	return nil
}
