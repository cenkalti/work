package inbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Message struct {
	Project   string    `json:"project"`
	Branch    string    `json:"branch,omitempty"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Name returns the display identifier: "<project>" at root, "<project>/<branch>" in worktree.
func (m *Message) Name() string {
	if m.Branch == "" {
		return m.Project
	}
	return m.Project + "/" + m.Branch
}

// Dir returns the global inbox directory: ~/.work/inbox/
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".work", "inbox"), nil
}

func filePath(sessionID string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sessionID+".json"), nil
}

func Write(msg *Message) error {
	if msg.SessionID == "" {
		return fmt.Errorf("missing session_id")
	}
	path, err := filePath(msg.SessionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func List() ([]*Message, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var msgs []*Message
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		msgPath := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(msgPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", msgPath, err)
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", msgPath, err)
		}
		msgs = append(msgs, &msg)
	}
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Timestamp.After(msgs[j].Timestamp)
	})
	return msgs, nil
}

func Delete(sessionID string) error {
	path, err := filePath(sessionID)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
