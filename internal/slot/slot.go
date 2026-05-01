// Package slot manages ~/.work/slots.json, the slot-index → agent-UUID map.
//
// The TUI is the only writer in production. agent rm clears stale entries.
// JSON keys are stringified ints (Go's encoding/json doesn't allow int map
// keys directly).
package slot

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

// Map maps slot index (1..9) to agent UUID.
type Map map[int]string

// Path returns ~/.work/slots.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".work", "slots.json"), nil
}

// Read returns the current slot map. A missing file yields an empty map.
func Read() (Map, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Map{}, nil
		}
		return nil, err
	}
	raw := map[string]string{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make(Map, len(raw))
	for k, v := range raw {
		n, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		out[n] = v
	}
	return out, nil
}

// Write atomically writes the slot map.
func Write(m Map) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	raw := make(map[string]string, len(m))
	for k, v := range m {
		raw[strconv.Itoa(k)] = v
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".slots.*.tmp")
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

// Set assigns slot to uuid, removing any prior assignment of the same slot or
// any prior slot held by uuid (one slot per agent, one agent per slot).
func Set(slot int, uuid string) error {
	m, err := Read()
	if err != nil {
		return err
	}
	for k, v := range m {
		if v == uuid {
			delete(m, k)
		}
	}
	m[slot] = uuid
	return Write(m)
}

// Clear removes the assignment at slot, if any.
func Clear(slot int) error {
	m, err := Read()
	if err != nil {
		return err
	}
	if _, ok := m[slot]; !ok {
		return nil
	}
	delete(m, slot)
	return Write(m)
}

// ClearByUUID removes any slot assignment for the given UUID.
func ClearByUUID(uuid string) error {
	m, err := Read()
	if err != nil {
		return err
	}
	changed := false
	for k, v := range m {
		if v == uuid {
			delete(m, k)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return Write(m)
}

// FindByUUID returns the slot holding uuid, or 0,false.
func FindByUUID(uuid string) (int, bool) {
	m, err := Read()
	if err != nil {
		return 0, false
	}
	for k, v := range m {
		if v == uuid {
			return k, true
		}
	}
	return 0, false
}
