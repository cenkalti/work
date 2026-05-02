package dash

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// sessionCache caches scanned session names keyed by session ID. The mtime is
// the JSONL file's modification time at scan; entries are reused while mtime
// is unchanged.
var (
	sessionCacheMu sync.Mutex
	sessionCache   = map[string]sessionCacheEntry{}
)

type sessionCacheEntry struct {
	name  string
	mtime time.Time
}

// claudeSessionName returns the latest session name for a Claude session:
// prefers a user-set custom-title (from /rename) and falls back to the
// auto-generated ai-title. Empty when no name is recorded or the session
// JSONL cannot be located.
func claudeSessionName(worktreePath, sessionID string) string {
	if sessionID == "" || worktreePath == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	enc := encodeProjectDir(worktreePath)
	file := filepath.Join(home, ".claude", "projects", enc, sessionID+".jsonl")
	info, err := os.Stat(file)
	if err != nil {
		return ""
	}
	sessionCacheMu.Lock()
	e, ok := sessionCache[sessionID]
	sessionCacheMu.Unlock()
	if ok && e.mtime.Equal(info.ModTime()) {
		return e.name
	}
	name := scanSessionName(file)
	sessionCacheMu.Lock()
	sessionCache[sessionID] = sessionCacheEntry{name: name, mtime: info.ModTime()}
	sessionCacheMu.Unlock()
	return name
}

// encodeProjectDir mirrors Claude Code's project-dir naming: each "/" or "."
// in the cwd becomes "-".
func encodeProjectDir(p string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(p)
}

func scanSessionName(file string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()
	var custom, ai string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<16), 1<<24)
	customMarker := []byte(`"type":"custom-title"`)
	aiMarker := []byte(`"type":"ai-title"`)
	for sc.Scan() {
		line := sc.Bytes()
		switch {
		case bytes.Contains(line, customMarker):
			var v struct {
				CustomTitle string `json:"customTitle"`
			}
			if json.Unmarshal(line, &v) == nil && v.CustomTitle != "" {
				custom = v.CustomTitle
			}
		case bytes.Contains(line, aiMarker):
			var v struct {
				AITitle string `json:"aiTitle"`
			}
			if json.Unmarshal(line, &v) == nil && v.AITitle != "" {
				ai = v.AITitle
			}
		}
	}
	if custom != "" {
		return custom
	}
	return ai
}
