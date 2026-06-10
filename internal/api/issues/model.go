// Package issues is the core domain: Linear-style issues with per-project
// numbering (TD-12), status, priority, assignee, and labels.
package issues

import (
	"fmt"
	"time"

	"github.com/Saurav-Paul/taskdock/internal/api/labels"
	"github.com/Saurav-Paul/taskdock/internal/api/projects"
	"github.com/Saurav-Paul/taskdock/internal/api/users"
)

// Valid status and priority values — mirror the CHECK constraints in the schema.
var (
	ValidStatuses  = []string{"backlog", "todo", "in_progress", "done", "canceled"}
	ValidPriorities = []string{"none", "low", "medium", "high", "urgent"}
)

// Issue is the GORM model for the "issues" table.
type Issue struct {
	ID          uint   `gorm:"primaryKey"`
	ProjectID   uint   `gorm:"not null"`
	Project     projects.Project
	Number      int    `gorm:"not null"`
	Title       string `gorm:"not null"`
	Description string `gorm:"not null;default:''"`
	Status      string `gorm:"not null;default:'todo'"`
	Priority    string `gorm:"not null;default:'none'"`
	AssigneeID  *uint
	Assignee    *users.User
	Labels      []labels.Label `gorm:"many2many:issue_labels"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Issue) TableName() string { return "issues" }

// Key returns the public identifier, e.g. "TD-12".
func (i *Issue) Key() string {
	return fmt.Sprintf("%s-%d", i.Project.Key, i.Number)
}

// IsValidStatus reports whether s is one of the allowed status values.
func IsValidStatus(s string) bool {
	for _, v := range ValidStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// IsValidPriority reports whether p is one of the allowed priority values.
func IsValidPriority(p string) bool {
	for _, v := range ValidPriorities {
		if v == p {
			return true
		}
	}
	return false
}
