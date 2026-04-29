package todo

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	orderFile  = "_order.json"
	archiveDir = ".archive"
	trashDir   = ".trash"
)

const idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

const idLen = 6

// Dir returns ~/.work/todos/.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".work", "todos"), nil
}

func openPath(dir, id string) string    { return filepath.Join(dir, id+".json") }
func archivePath(dir, id string) string { return filepath.Join(dir, archiveDir, id+".json") }
func trashPath(dir, id string) string   { return filepath.Join(dir, trashDir, id+".json") }
func orderPath(dir string) string       { return filepath.Join(dir, orderFile) }

// EnsureDir creates the store directory if missing.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// Load reads a single todo's JSON file from the open store.
func Load(dir, id string) (*Todo, error) {
	data, err := os.ReadFile(openPath(dir, id))
	if err != nil {
		return nil, err
	}
	var t Todo
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", id, err)
	}
	return &t, nil
}

// Write writes a todo to the open store, atomically via tempfile + rename.
func Write(dir string, t *Todo) error {
	if err := EnsureDir(dir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(openPath(dir, t.ID), data)
}

// Trash moves <id>.json into .trash/.
func Trash(dir, id string) error {
	return moveTo(dir, id, trashDir)
}

// Archive moves <id>.json into .archive/.
func Archive(dir, id string) error {
	return moveTo(dir, id, archiveDir)
}

func moveTo(dir, id, sub string) error {
	subdir := filepath.Join(dir, sub)
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		return err
	}
	src := openPath(dir, id)
	dst := filepath.Join(subdir, id+".json")
	return os.Rename(src, dst)
}

// LoadAll reads every open todo and returns them keyed by id along with the
// canonical top-level order. If _order.json is missing, references missing
// ids, or omits ids that exist on disk, the order is repaired and a warning
// is printed to stderr.
func LoadAll(dir string) (map[string]*Todo, []string, error) {
	if err := EnsureDir(dir); err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	todos := make(map[string]*Todo)
	mtimes := make(map[string]int64)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || name == orderFile {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		t, err := Load(dir, id)
		if err != nil {
			return nil, nil, err
		}
		todos[id] = t
		info, err := e.Info()
		if err != nil {
			return nil, nil, err
		}
		mtimes[id] = info.ModTime().UnixNano()
	}

	order, repaired := loadOrder(dir, todos, mtimes)
	if repaired && len(todos) > 0 {
		fmt.Fprintln(os.Stderr, "todo: rebuilt _order.json from filesystem")
	}
	return todos, order, nil
}

func loadOrder(dir string, todos map[string]*Todo, mtimes map[string]int64) ([]string, bool) {
	repaired := false
	var stored []string
	data, err := os.ReadFile(orderPath(dir))
	if err == nil {
		_ = json.Unmarshal(data, &stored)
	} else {
		repaired = true
	}

	// Top-level = ids that aren't a child of any other todo.
	childOf := make(map[string]bool)
	for _, t := range todos {
		for _, c := range t.Children {
			childOf[c] = true
		}
	}

	var order []string
	seen := make(map[string]bool)
	for _, id := range stored {
		if _, ok := todos[id]; !ok {
			repaired = true
			continue
		}
		if childOf[id] {
			repaired = true
			continue
		}
		if seen[id] {
			repaired = true
			continue
		}
		seen[id] = true
		order = append(order, id)
	}

	// Append any top-level ids missing from stored, mtime descending.
	var missing []string
	for id := range todos {
		if seen[id] || childOf[id] {
			continue
		}
		missing = append(missing, id)
	}
	if len(missing) > 0 {
		repaired = true
		sort.Slice(missing, func(i, j int) bool {
			return mtimes[missing[i]] > mtimes[missing[j]]
		})
		order = append(order, missing...)
	}

	return order, repaired
}

// WriteOrder writes _order.json atomically via tempfile + rename.
func WriteOrder(dir string, order []string) error {
	if err := EnsureDir(dir); err != nil {
		return err
	}
	if order == nil {
		order = []string{}
	}
	data, err := json.MarshalIndent(order, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(orderPath(dir), data)
}

// GenerateID returns a 6-char [a-z0-9] id that doesn't collide with anything
// in the open store, .archive/, or .trash/.
func GenerateID(dir string) (string, error) {
	for range 1000 {
		id, err := randomID()
		if err != nil {
			return "", err
		}
		if idTaken(dir, id) {
			continue
		}
		return id, nil
	}
	return "", fmt.Errorf("could not generate unique id after 1000 attempts")
}

func idTaken(dir, id string) bool {
	for _, p := range []string{openPath(dir, id), archivePath(dir, id), trashPath(dir, id)} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func randomID() (string, error) {
	buf := make([]byte, idLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, idLen)
	for i, b := range buf {
		out[i] = idAlphabet[int(b)%len(idAlphabet)]
	}
	return string(out), nil
}

func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
