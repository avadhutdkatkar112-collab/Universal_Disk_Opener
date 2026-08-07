package storage_test

import (
	"context"
	"testing"

	"github.com/user/vhd-opener/pkg/storage"
)

func TestEvidenceSession_Lifecycle(t *testing.T) {
	session := storage.NewEvidenceSession("test-001")

	if session.ID() != "test-001" {
		t.Errorf("Expected session ID test-001, got %s", session.ID())
	}

	if session.State() != storage.SessionClosed {
		t.Errorf("Expected initial state SessionClosed, got %d", session.State())
	}

	ctx := context.Background()
	err := session.Open(ctx, "/path/to/evidence.raw")
	if err != nil {
		t.Fatalf("Failed to open session: %v", err)
	}

	if session.State() != storage.SessionOpen {
		t.Errorf("Expected state SessionOpen, got %d", session.State())
	}

	err = session.Close()
	if err != nil {
		t.Fatalf("Failed to close session: %v", err)
	}

	if session.State() != storage.SessionClosed {
		t.Errorf("Expected state SessionClosed after close, got %d", session.State())
	}
}

func TestEvidenceSession_Provenance(t *testing.T) {
	session := storage.NewEvidenceSession("test-002")
	ctx := context.Background()

	session.Open(ctx, "/path/to/evidence.raw")
	session.LogProvenance("analyst", "file.view", "/Windows/System32", "Viewed directory")
	session.LogProvenance("analyst", "bookmark.add", "/Windows/System32/config/SYSTEM", "Bookmarked SYSTEM hive")

	provenance := session.Provenance()

	if len(provenance) != 3 {
		t.Fatalf("Expected 3 provenance entries, got %d", len(provenance))
	}

	if provenance[0].Action != "session.open" {
		t.Errorf("Expected first action session.open, got %s", provenance[0].Action)
	}

	if provenance[1].Action != "file.view" {
		t.Errorf("Expected second action file.view, got %s", provenance[1].Action)
	}

	if provenance[2].Target != "/Windows/System32/config/SYSTEM" {
		t.Errorf("Expected third target /Windows/System32/config/SYSTEM, got %s", provenance[2].Target)
	}

	for _, entry := range provenance {
		if entry.SessionID != "test-002" {
			t.Errorf("Expected session ID test-002 in provenance, got %s", entry.SessionID)
		}
		if entry.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp in provenance")
		}
	}
}

func TestEvidenceSession_ListDirectory_NoFilesystem(t *testing.T) {
	session := storage.NewEvidenceSession("test-003")
	ctx := context.Background()

	session.Open(ctx, "/path/to/evidence.raw")

	_, err := session.ListDirectory(ctx, "/")
	if err != storage.ErrNoFilesystemMounted {
		t.Errorf("Expected ErrNoFilesystemMounted, got: %v", err)
	}
}

func TestEvidenceSession_Metadata(t *testing.T) {
	session := storage.NewEvidenceSession("test-004")
	ctx := context.Background()

	session.Open(ctx, "/path/to/evidence.raw")

	meta := session.Metadata()
	if meta.ImagePath != "/path/to/evidence.raw" {
		t.Errorf("Expected image path /path/to/evidence.raw, got %s", meta.ImagePath)
	}
}

func TestEvidenceSession_ThreadSafety(t *testing.T) {
	session := storage.NewEvidenceSession("test-005")
	ctx := context.Background()

	session.Open(ctx, "/path/to/evidence.raw")

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			session.LogProvenance("analyst", "concurrent.action", "target", "details")
			session.State()
			session.Metadata()
			session.Provenance()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	provenance := session.Provenance()
	if len(provenance) < 10 {
		t.Errorf("Expected at least 10 provenance entries from concurrent writes, got %d", len(provenance))
	}
}
