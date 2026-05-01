package slot

import (
	"path/filepath"
	"testing"
)

func setupTempHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestReadMissingReturnsEmpty(t *testing.T) {
	setupTempHome(t)
	m, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Errorf("want empty map, got %v", m)
	}
}

func TestWriteRead(t *testing.T) {
	setupTempHome(t)

	want := Map{1: "uuid-a", 3: "uuid-b"}
	if err := Write(want); err != nil {
		t.Fatal(err)
	}
	got, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != "uuid-a" || got[3] != "uuid-b" {
		t.Errorf("round-trip mismatch: %v", got)
	}
}

func TestSetClear(t *testing.T) {
	setupTempHome(t)

	if err := Set(1, "uuid-a"); err != nil {
		t.Fatal(err)
	}
	if err := Set(3, "uuid-b"); err != nil {
		t.Fatal(err)
	}
	m, _ := Read()
	if m[1] != "uuid-a" || m[3] != "uuid-b" {
		t.Fatalf("after Set: %v", m)
	}

	if err := Clear(1); err != nil {
		t.Fatal(err)
	}
	m, _ = Read()
	if _, ok := m[1]; ok {
		t.Errorf("slot 1 still present: %v", m)
	}
	if m[3] != "uuid-b" {
		t.Errorf("slot 3 lost: %v", m)
	}

	// Clear of missing slot is a no-op.
	if err := Clear(7); err != nil {
		t.Errorf("Clear missing: %v", err)
	}
}

func TestSetMovesAgent(t *testing.T) {
	setupTempHome(t)
	if err := Set(1, "uuid-a"); err != nil {
		t.Fatal(err)
	}
	if err := Set(3, "uuid-a"); err != nil {
		t.Fatal(err)
	}
	m, _ := Read()
	if _, ok := m[1]; ok {
		t.Errorf("slot 1 should be cleared after moving uuid-a to slot 3: %v", m)
	}
	if m[3] != "uuid-a" {
		t.Errorf("slot 3 missing uuid-a: %v", m)
	}
}

func TestSetDisplacesPriorOccupant(t *testing.T) {
	setupTempHome(t)
	_ = Set(1, "uuid-a")
	_ = Set(1, "uuid-b")
	m, _ := Read()
	if m[1] != "uuid-b" {
		t.Errorf("slot 1 should be uuid-b: %v", m)
	}
}

func TestClearByUUID(t *testing.T) {
	setupTempHome(t)
	_ = Set(1, "uuid-a")
	_ = Set(3, "uuid-b")
	if err := ClearByUUID("uuid-a"); err != nil {
		t.Fatal(err)
	}
	m, _ := Read()
	if _, ok := m[1]; ok {
		t.Errorf("slot 1 should be cleared: %v", m)
	}
	if m[3] != "uuid-b" {
		t.Errorf("slot 3 disturbed: %v", m)
	}
	if err := ClearByUUID("nonexistent"); err != nil {
		t.Errorf("ClearByUUID on missing: %v", err)
	}
}

func TestFindByUUID(t *testing.T) {
	setupTempHome(t)
	_ = Set(2, "uuid-a")
	if slot, ok := FindByUUID("uuid-a"); !ok || slot != 2 {
		t.Errorf("want (2,true), got (%d,%v)", slot, ok)
	}
	if _, ok := FindByUUID("nope"); ok {
		t.Errorf("want false for missing uuid")
	}
}

func TestPath(t *testing.T) {
	setupTempHome(t)
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "slots.json" {
		t.Errorf("unexpected basename: %s", p)
	}
}
