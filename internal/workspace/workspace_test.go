package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func cleanDir(t *testing.T) string {
	t.Helper()
	dir, _ := os.MkdirTemp("", "workspace-test")
	os.RemoveAll(filepath.Join(dir, "workspace"))
	return dir
}

func TestWorkspace_OpenDisk(t *testing.T) {
	dir := cleanDir(t)
	defer os.RemoveAll(dir)

	ws := NewWorkspace(dir)

	disk := ws.OpenDisk("/path/to/disk.vhd")
	if disk == nil {
		t.Fatal("Expected non-nil disk session")
	}
	if disk.Path != "/path/to/disk.vhd" {
		t.Errorf("Expected path /path/to/disk.vhd, got %s", disk.Path)
	}
	if disk.Name != "disk.vhd" {
		t.Errorf("Expected name disk.vhd, got %s", disk.Name)
	}

	disks := ws.ListDisks()
	if len(disks) != 1 {
		t.Fatalf("Expected 1 disk, got %d", len(disks))
	}

	if ws.GetActiveDiskID() != disk.ID {
		t.Errorf("Expected active disk %s, got %s", disk.ID, ws.GetActiveDiskID())
	}
}

func TestWorkspace_OpenMultipleDisks(t *testing.T) {
	dir := cleanDir(t)
	defer os.RemoveAll(dir)

	ws := NewWorkspace(dir)

	ws.OpenDisk("/disk1.vhd")
	ws.OpenDisk("/disk2.vmdk")
	ws.OpenDisk("/disk3.qcow2")

	disks := ws.ListDisks()
	if len(disks) != 3 {
		t.Fatalf("Expected 3 disks, got %d", len(disks))
	}

	ws.SetActiveDisk(disks[0].ID)
	if ws.GetActiveDiskID() != disks[0].ID {
		t.Errorf("Expected active disk %s, got %s", disks[0].ID, ws.GetActiveDiskID())
	}
}

func TestWorkspace_CloseDisk(t *testing.T) {
	dir := cleanDir(t)
	defer os.RemoveAll(dir)

	ws := NewWorkspace(dir)

	d1 := ws.OpenDisk("/disk1.vhd")
	ws.OpenDisk("/disk2.vmdk")

	ws.CloseDisk(d1.ID)

	disks := ws.ListDisks()
	if len(disks) != 1 {
		t.Fatalf("Expected 1 disk after close, got %d", len(disks))
	}
}

func TestWorkspace_Navigate(t *testing.T) {
	dir := cleanDir(t)
	defer os.RemoveAll(dir)

	ws := NewWorkspace(dir)
	disk := ws.OpenDisk("/disk.vhd")

	ws.Navigate(disk.ID, "/Windows/System32")
	ws.Navigate(disk.ID, "/Windows/System32/config")

	d, _ := ws.GetDisk(disk.ID)
	if d.CurrentPath != "/Windows/System32/config" {
		t.Errorf("Expected current path /Windows/System32/config, got %s", d.CurrentPath)
	}

	history := ws.GetHistory(disk.ID)
	if len(history) != 2 {
		t.Fatalf("Expected 2 history entries, got %d", len(history))
	}
}

func TestWorkspace_Bookmarks(t *testing.T) {
	dir := cleanDir(t)
	defer os.RemoveAll(dir)

	ws := NewWorkspace(dir)
	disk := ws.OpenDisk("/disk.vhd")

	ws.AddBookmark(disk.ID, "System32", "/Windows/System32", "Core system files")

	ws.AddBookmark(disk.ID, "Drivers", "/Windows/System32/drivers", "Driver files")

	bookmarks := ws.GetBookmarks(disk.ID)
	if len(bookmarks) != 2 {
		t.Fatalf("Expected 2 bookmarks, got %d", len(bookmarks))
	}

	ws.RemoveBookmark(disk.ID, bookmarks[0].ID)
	bookmarks = ws.GetBookmarks(disk.ID)
	if len(bookmarks) != 1 {
		t.Fatalf("Expected 1 bookmark after remove, got %d", len(bookmarks))
	}
}

func TestWorkspace_Partition(t *testing.T) {
	dir := cleanDir(t)
	defer os.RemoveAll(dir)

	ws := NewWorkspace(dir)
	disk := ws.OpenDisk("/disk.vhd")

	ws.SetPartition(disk.ID, 2)

	d, _ := ws.GetDisk(disk.ID)
	if d.ActivePartition != 2 {
		t.Errorf("Expected partition 2, got %d", d.ActivePartition)
	}
	if d.CurrentPath != "/" {
		t.Errorf("Expected path reset to /, got %s", d.CurrentPath)
	}
}

func TestWorkspace_Persistence(t *testing.T) {
	dir := cleanDir(t)
	defer os.RemoveAll(dir)

	ws := NewWorkspace(dir)
	ws.OpenDisk("/disk1.vhd")
	ws.OpenDisk("/disk2.vmdk")

	ws2 := NewWorkspace(dir)
	disks := ws2.ListDisks()
	if len(disks) != 2 {
		t.Fatalf("Expected 2 disks after reload, got %d", len(disks))
	}
}
