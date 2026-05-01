package dash

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/slot"
	"github.com/google/uuid"
)

func setupTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func writeAgent(t *testing.T, name, sessionID string, status string, lastActivity time.Time) *agent.Record {
	t.Helper()
	id, _ := uuid.NewV7()
	now := time.Now().UTC()
	r := &agent.Record{
		ID:               id.String(),
		Name:             name,
		Project:          "mux",
		ProjectRoot:      "/tmp/mux",
		TaskID:           name,
		Branch:           name,
		WorktreePath:     filepath.Join(t.TempDir(), name),
		CurrentSessionID: sessionID,
		Status:           status,
		CreatedAt:        now,
		UpdatedAt:        now,
		LastActivity:     lastActivity,
	}
	// ensure worktree path exists so NoWorktree stays false unless we want it
	_ = os.MkdirAll(r.WorktreePath, 0o755)
	if err := agent.Write(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestLoadRowsSortAndSlot(t *testing.T) {
	setupTempHome(t)

	old := writeAgent(t, "old", "sess-old", agent.StatusIdle, time.Now().Add(-10*time.Minute))
	mid := writeAgent(t, "mid", "sess-mid", agent.StatusIdle, time.Now().Add(-5*time.Minute))
	recent := writeAgent(t, "recent", "sess-recent", agent.StatusIdle, time.Now().Add(-1*time.Minute))
	pinned := writeAgent(t, "pinned", "sess-pinned", agent.StatusIdle, time.Now().Add(-1*time.Hour))

	if err := slot.Set(2, pinned.ID); err != nil {
		t.Fatal(err)
	}

	rows, err := loadRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("want 4 rows, got %d", len(rows))
	}
	// pinned with slot=2 should come first.
	if rows[0].AgentID != pinned.ID || rows[0].Slot != 2 || !rows[0].HasSlot {
		t.Errorf("row 0: want pinned in slot 2, got id=%s slot=%d hasSlot=%v", rows[0].AgentID, rows[0].Slot, rows[0].HasSlot)
	}
	// then unassigned by last_activity desc: recent, mid, old
	if rows[1].AgentID != recent.ID {
		t.Errorf("row 1: want recent, got %s", rows[1].AgentID)
	}
	if rows[2].AgentID != mid.ID {
		t.Errorf("row 2: want mid, got %s", rows[2].AgentID)
	}
	if rows[3].AgentID != old.ID {
		t.Errorf("row 3: want old, got %s", rows[3].AgentID)
	}
}

func TestLoadRowsNoWorktreeAndCrashed(t *testing.T) {
	setupTempHome(t)

	// Worktree directory does NOT exist for this one.
	id1, _ := uuid.NewV7()
	rec := &agent.Record{
		ID:               id1.String(),
		Name:             "ghost",
		Project:          "mux",
		Status:           agent.StatusRunning,
		CurrentSessionID: "ghost-session",
		WorktreePath:     "/this/path/does/not/exist",
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := agent.Write(rec); err != nil {
		t.Fatal(err)
	}

	rows, err := loadRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if !rows[0].NoWorktree {
		t.Error("expected NoWorktree=true")
	}
	if !rows[0].Crashed {
		t.Error("expected Crashed=true (status=running, no session file)")
	}
}

func TestLoadRowsAliveSessionNotCrashed(t *testing.T) {
	home := setupTempHome(t)

	rec := writeAgent(t, "live", "live-session", agent.StatusRunning, time.Now())

	// Drop a fake claude session file.
	sessDir := filepath.Join(home, ".claude", "sessions")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"pid":1234,"sessionId":"live-session","status":"busy"}`)
	if err := os.WriteFile(filepath.Join(sessDir, "1234.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	rows, err := loadRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].Crashed {
		t.Errorf("expected Crashed=false; rec=%+v", rec)
	}
}
