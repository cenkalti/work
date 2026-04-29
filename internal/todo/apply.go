package todo

import (
	"fmt"
	"os"
	"reflect"
	"time"
)

// Apply takes a parsed buffer plus the on-disk snapshot and writes back the
// diff: new items get ids, removed items go to .trash/, modified items are
// rewritten, parents' Children lists are recomputed, status transitions
// update ClosedAt, and _order.json reflects the new top-level order.
func Apply(dir string, parsed []*ParsedTodo, snapshot map[string]*Todo, now time.Time) error {
	// Pass 0: validate hand-pinned ids that aren't in the open snapshot don't
	// collide with archived or trashed entries. Reject before any writes.
	if err := validateHandPinned(dir, parsed, snapshot); err != nil {
		return err
	}

	// Pass 1: assign ids to items without one.
	if err := assignIDs(dir, parsed); err != nil {
		return err
	}

	// Pass 2: build the new id->Todo map and the new top-level order.
	newTodos := make(map[string]*Todo)
	for _, p := range parsed {
		buildTodos(p, snapshot, now, newTodos)
	}
	newOrder := make([]string, len(parsed))
	for i, p := range parsed {
		newOrder[i] = p.ID
	}

	// Pass 3: writes for new and changed items.
	for id, t := range newTodos {
		old, ok := snapshot[id]
		if !ok || !reflect.DeepEqual(old, t) {
			if err := Write(dir, t); err != nil {
				return err
			}
		}
	}

	// Pass 4: order file.
	if err := WriteOrder(dir, newOrder); err != nil {
		return err
	}

	// Pass 5: trash items that disappeared.
	for id := range snapshot {
		if _, ok := newTodos[id]; ok {
			continue
		}
		if err := Trash(dir, id); err != nil {
			return err
		}
	}

	return nil
}

func validateHandPinned(dir string, parsed []*ParsedTodo, snapshot map[string]*Todo) error {
	var walk func(items []*ParsedTodo) error
	walk = func(items []*ParsedTodo) error {
		for _, p := range items {
			if p.ID != "" {
				if _, inOpen := snapshot[p.ID]; !inOpen {
					if existsAt(archivePath(dir, p.ID)) {
						return fmt.Errorf("line %d: id %q collides with an archived item; remove the <!--%s--> stamp to assign a fresh id", p.Line, p.ID, p.ID)
					}
					if existsAt(trashPath(dir, p.ID)) {
						return fmt.Errorf("line %d: id %q collides with a trashed item; remove the <!--%s--> stamp to assign a fresh id", p.Line, p.ID, p.ID)
					}
				}
			}
			if err := walk(p.Children); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(parsed)
}

func existsAt(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func assignIDs(dir string, items []*ParsedTodo) error {
	for _, p := range items {
		if p.ID == "" {
			id, err := GenerateID(dir)
			if err != nil {
				return err
			}
			p.ID = id
		}
		if err := assignIDs(dir, p.Children); err != nil {
			return err
		}
	}
	return nil
}

// buildTodos converts a ParsedTodo (and its descendants) into Todo structs,
// inserting them into out keyed by id. ClosedAt is computed by comparing
// the new status against the snapshot's status.
func buildTodos(p *ParsedTodo, snapshot map[string]*Todo, now time.Time, out map[string]*Todo) {
	t := &Todo{
		ID:       p.ID,
		Title:    p.Title,
		Status:   p.Status,
		Links:    p.Links,
		Projects: p.Projects,
		Notes:    p.Notes,
	}
	for _, c := range p.Children {
		t.Children = append(t.Children, c.ID)
	}
	t.ClosedAt = nextClosedAt(snapshot[p.ID], p.Status, now)
	out[p.ID] = t
	for _, c := range p.Children {
		buildTodos(c, snapshot, now, out)
	}
}

func nextClosedAt(old *Todo, newStatus string, now time.Time) *time.Time {
	newClosed := IsClosed(newStatus)
	if old == nil {
		if newClosed {
			n := now
			return &n
		}
		return nil
	}
	oldClosed := IsClosed(old.Status)
	switch {
	case !oldClosed && newClosed:
		n := now
		return &n
	case oldClosed && !newClosed:
		return nil
	default:
		return old.ClosedAt
	}
}
