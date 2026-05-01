// Package order manages ~/.work/order.json, the user-defined display order
// of agent UUIDs in the dashboard.
//
// The TUI is the only writer. The list is normalized on every dashboard load:
// missing UUIDs are pruned, new agents are appended.
package order

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Path returns ~/.work/order.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".work", "order.json"), nil
}

// Read returns the persisted order. A missing file yields an empty slice.
func Read() ([]string, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	var out []string
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Write atomically writes the order list.
func Write(o []string) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".order.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, p)
}

// Swap exchanges the positions of a and b in the persisted order. Either UUID
// missing from the list is appended before the swap. Returns true if the file
// was changed.
func Swap(a, b string) (bool, error) {
	if a == "" || b == "" || a == b {
		return false, nil
	}
	o, err := Read()
	if err != nil {
		return false, err
	}
	ia, ib := -1, -1
	for i, v := range o {
		if v == a {
			ia = i
		}
		if v == b {
			ib = i
		}
	}
	if ia == -1 {
		o = append(o, a)
		ia = len(o) - 1
	}
	if ib == -1 {
		o = append(o, b)
		ib = len(o) - 1
	}
	o[ia], o[ib] = o[ib], o[ia]
	if err := Write(o); err != nil {
		return false, err
	}
	return true, nil
}
