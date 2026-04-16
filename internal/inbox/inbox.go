package inbox

import (
	"encoding/json"
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

func filePath(project, branch string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	leaf := "_.json"
	if branch != "" {
		leaf = branch + ".json"
	}
	return filepath.Join(dir, project, leaf), nil
}

func Write(msg *Message) error {
	path, err := filePath(msg.Project, msg.Branch)
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
	projectDirs, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var msgs []*Message
	for _, pd := range projectDirs {
		if !pd.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(dir, pd.Name()))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, pd.Name(), e.Name()))
			if err != nil {
				continue
			}
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			msgs = append(msgs, &msg)
		}
	}
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Timestamp.After(msgs[j].Timestamp)
	})
	return msgs, nil
}

func Delete(project, branch string) error {
	path, err := filePath(project, branch)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
