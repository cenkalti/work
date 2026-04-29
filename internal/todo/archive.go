package todo

import (
	"slices"
	"time"
)

const archiveAfter = 14 * 24 * time.Hour

// ArchiveSweep moves closed todos (completed or cancelled) whose ClosedAt is
// older than 14 days from the open store into .archive/. A todo whose
// subtree contains a still-live (non-archivable) item is kept in place.
// Idempotent: running twice in a row is a no-op the second time.
func ArchiveSweep(dir string, now time.Time) error {
	todos, order, err := LoadAll(dir)
	if err != nil {
		return err
	}

	archivable := func(id string) bool {
		t, ok := todos[id]
		if !ok {
			return false
		}
		if !IsClosed(t.Status) || t.ClosedAt == nil {
			return false
		}
		return now.Sub(*t.ClosedAt) > archiveAfter
	}

	var subtreeArchivable func(id string) bool
	subtreeArchivable = func(id string) bool {
		if !archivable(id) {
			return false
		}
		t := todos[id]
		for _, c := range t.Children {
			if !subtreeArchivable(c) {
				return false
			}
		}
		return true
	}

	var toArchive []string
	var collect func(id string)
	collect = func(id string) {
		t, ok := todos[id]
		if !ok {
			return
		}
		for _, c := range t.Children {
			collect(c)
		}
		if subtreeArchivable(id) {
			toArchive = append(toArchive, id)
		}
	}
	for _, id := range order {
		collect(id)
	}

	if len(toArchive) == 0 {
		return nil
	}

	archiving := make(map[string]bool, len(toArchive))
	for _, id := range toArchive {
		archiving[id] = true
	}

	// Rewrite parents whose children list contained an archived id.
	parentOf := make(map[string]string)
	for pid, t := range todos {
		for _, cid := range t.Children {
			parentOf[cid] = pid
		}
	}
	parentsToRewrite := make(map[string]bool)
	for id := range archiving {
		if pid, ok := parentOf[id]; ok && !archiving[pid] {
			parentsToRewrite[pid] = true
		}
	}
	for pid := range parentsToRewrite {
		t := todos[pid]
		filtered := t.Children[:0:0]
		for _, cid := range t.Children {
			if !archiving[cid] {
				filtered = append(filtered, cid)
			}
		}
		t.Children = filtered
		if err := Write(dir, t); err != nil {
			return err
		}
	}

	// Rewrite _order.json if any top-level item was archived.
	newOrder := order[:0:0]
	for _, id := range order {
		if !archiving[id] {
			newOrder = append(newOrder, id)
		}
	}
	if !slices.Equal(newOrder, order) {
		if err := WriteOrder(dir, newOrder); err != nil {
			return err
		}
	}

	for _, id := range toArchive {
		if err := Archive(dir, id); err != nil {
			return err
		}
	}

	return nil
}
