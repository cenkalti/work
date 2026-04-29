package todo

import "time"

const (
	StatusOpen      = "open"
	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

type Todo struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Status   string     `json:"status"`
	ClosedAt *time.Time `json:"closed_at"`
	Links    []Link     `json:"links,omitempty"`
	Projects []string   `json:"projects,omitempty"`
	Notes    string     `json:"notes,omitempty"`
	Children []string   `json:"children,omitempty"`
}

type Link struct {
	Label string `json:"label,omitempty"`
	URL   string `json:"url"`
}

func IsClosed(status string) bool {
	return status == StatusCompleted || status == StatusCancelled
}
