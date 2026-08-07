package hash

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/user/vhd-opener/pkg/capability"
)

// NIST CAVP Test Vectors (FIPS 180-4, FIPS 180-2)
// https://csrc.nist.gov/projects/cryptographic-algorithm-validation-program

func makeCtx(params map[string]any) capability.ExecutionContext {
	return capability.ExecutionContext{Params: params}
}

func TestNISTEmptyString(t *testing.T) {
	// SHA-256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	expectedSHA256 := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	// SHA-1("") = da39a3ee5e6b4b0d3255bfef95601890afd80709
	expectedSHA1 := "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	// MD5("") = d41d8cd98f00b204e9800998ecf8427e
	expectedMD5 := "d41d8cd98f00b204e9800998ecf8427e"

	readFile := func(path string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("")), nil
	}

	cap := NewHashingCapability(readFile)
	execCtx := makeCtx(map[string]any{"path": "/test/empty.txt"})

	result, err := cap.Execute(context.Background(), execCtx, make(chan float64, 100))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	hashResult, ok := result.(HashResult)
	if !ok {
		t.Fatalf("Expected HashResult, got %T", result)
	}

	if hashResult.Hashes["sha256"] != expectedSHA256 {
		t.Errorf("SHA-256 mismatch:\n  got:  %s\n  want: %s", hashResult.Hashes["sha256"], expectedSHA256)
	}
	if hashResult.Hashes["sha1"] != expectedSHA1 {
		t.Errorf("SHA-1 mismatch:\n  got:  %s\n  want: %s", hashResult.Hashes["sha1"], expectedSHA1)
	}
	if hashResult.Hashes["md5"] != expectedMD5 {
		t.Errorf("MD5 mismatch:\n  got:  %s\n  want: %s", hashResult.Hashes["md5"], expectedMD5)
	}
}

func TestNISTSixteenBytes(t *testing.T) {
	input := "0123456789abcdef"

	h256 := sha256.Sum256([]byte(input))
	expectedSHA256 := hex.EncodeToString(h256[:])

	h1 := sha1.Sum([]byte(input))
	expectedSHA1 := hex.EncodeToString(h1[:])

	hmd5 := md5.Sum([]byte(input))
	expectedMD5 := hex.EncodeToString(hmd5[:])

	readFile := func(path string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(input)), nil
	}

	cap := NewHashingCapability(readFile)
	execCtx := makeCtx(map[string]any{"path": "/test/data.bin"})

	result, err := cap.Execute(context.Background(), execCtx, make(chan float64, 100))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	hashResult := result.(HashResult)

	if hashResult.Hashes["sha256"] != expectedSHA256 {
		t.Errorf("SHA-256 mismatch:\n  got:  %s\n  want: %s", hashResult.Hashes["sha256"], expectedSHA256)
	}
	if hashResult.Hashes["sha1"] != expectedSHA1 {
		t.Errorf("SHA-1 mismatch:\n  got:  %s\n  want: %s", hashResult.Hashes["sha1"], expectedSHA1)
	}
	if hashResult.Hashes["md5"] != expectedMD5 {
		t.Errorf("MD5 mismatch:\n  got:  %s\n  want: %s", hashResult.Hashes["md5"], expectedMD5)
	}
	if hashResult.Size != int64(len(input)) {
		t.Errorf("Size mismatch: got %d, want %d", hashResult.Size, len(input))
	}
}

func TestNISTLargePayload(t *testing.T) {
	// 1 MB of 'A' bytes
	input := strings.Repeat("A", 1024*1024)

	h256 := sha256.Sum256([]byte(input))
	expectedSHA256 := hex.EncodeToString(h256[:])

	hmd5 := md5.Sum([]byte(input))
	expectedMD5 := hex.EncodeToString(hmd5[:])

	readFile := func(path string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(input)), nil
	}

	cap := NewHashingCapability(readFile)
	execCtx := makeCtx(map[string]any{"path": "/test/1mb.bin"})

	progressValues := make(chan float64, 1000)
	result, err := cap.Execute(context.Background(), execCtx, progressValues)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	hashResult := result.(HashResult)

	if hashResult.Hashes["sha256"] != expectedSHA256 {
		t.Errorf("SHA-256 mismatch for 1MB payload")
	}
	if hashResult.Hashes["md5"] != expectedMD5 {
		t.Errorf("MD5 mismatch for 1MB payload")
	}
	if hashResult.Size != 1024*1024 {
		t.Errorf("Size mismatch: got %d, want %d", hashResult.Size, 1024*1024)
	}
	if hashResult.ThroughputMBps <= 0 {
		t.Errorf("Throughput should be > 0, got %f", hashResult.ThroughputMBps)
	}
}

func TestHashVerification(t *testing.T) {
	input := "verify me"
	h := sha256.Sum256([]byte(input))
	correctHash := hex.EncodeToString(h[:])

	readFile := func(path string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(input)), nil
	}

	cap := NewHashingCapability(readFile)

	// Test MATCH_VERIFIED
	execCtx := makeCtx(map[string]any{"path": "/test/verify.txt", "verify_hash": correctHash})
	result, err := cap.Execute(context.Background(), execCtx, make(chan float64, 10))
	if err != nil {
		t.Fatal(err)
	}
	if result.(HashResult).MatchStatus != "MATCH_VERIFIED" {
		t.Errorf("Expected MATCH_VERIFIED, got %s", result.(HashResult).MatchStatus)
	}

	// Test MISMATCH
	execCtx2 := makeCtx(map[string]any{"path": "/test/verify.txt", "verify_hash": "0000000000000000000000000000000000000000000000000000000000000000"})
	result2, err := cap.Execute(context.Background(), execCtx2, make(chan float64, 10))
	if err != nil {
		t.Fatal(err)
	}
	if result2.(HashResult).MatchStatus != "MISMATCH" {
		t.Errorf("Expected MISMATCH, got %s", result2.(HashResult).MatchStatus)
	}

	// Test UNVERIFIED (no verify_hash)
	execCtx3 := makeCtx(map[string]any{"path": "/test/verify.txt"})
	result3, err := cap.Execute(context.Background(), execCtx3, make(chan float64, 10))
	if err != nil {
		t.Fatal(err)
	}
	if result3.(HashResult).MatchStatus != "NO_MATCH" {
		t.Errorf("Expected NO_MATCH, got %s", result3.(HashResult).MatchStatus)
	}
}

func TestContextCancellation(t *testing.T) {
	readFile := func(path string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("X"), 1024*1024))), nil
	}

	cap := NewHashingCapability(readFile)
	execCtx := makeCtx(map[string]any{"path": "/test/cancel.bin"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := cap.Execute(ctx, execCtx, make(chan float64, 10))
	if err == nil {
		t.Error("Expected error from cancelled context, got nil")
	}
}

func TestValidateRequiresPath(t *testing.T) {
	cap := NewHashingCapability(nil)

	// Empty path
	if err := cap.Validate(makeCtx(map[string]any{})); err == nil {
		t.Error("Expected error for empty params")
	}

	// Blank path
	if err := cap.Validate(makeCtx(map[string]any{"path": "  "})); err == nil {
		t.Error("Expected error for blank path")
	}

	// Valid path
	if err := cap.Validate(makeCtx(map[string]any{"path": "/valid/path"})); err != nil {
		t.Errorf("Unexpected error for valid path: %v", err)
	}
}

func TestThroughputCalculation(t *testing.T) {
	input := strings.Repeat("B", 4*1024*1024) // 4 MB

	readFile := func(path string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte(input))), nil
	}

	cap := NewHashingCapability(readFile)
	execCtx := makeCtx(map[string]any{"path": "/test/4mb.bin"})

	result, err := cap.Execute(context.Background(), execCtx, make(chan float64, 100))
	if err != nil {
		t.Fatal(err)
	}

	hr := result.(HashResult)
	if hr.ThroughputMBps <= 0 {
		t.Errorf("Throughput should be positive, got %f", hr.ThroughputMBps)
	}
	if hr.ElapsedSeconds <= 0 {
		t.Errorf("Elapsed should be positive, got %f", hr.ElapsedSeconds)
	}
}
