package issues

// Data access layer — all database queries for issues.

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/Saurav-Paul/taskdock/internal/api/labels"
	"github.com/Saurav-Paul/taskdock/internal/api/projects"
)

// Repository provides database operations for the issues table.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Repository with the given database connection.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// withRelations preloads everything needed to render an issue response.
// Related issues preload their own Project so their keys can be computed.
func (r *Repository) withRelations() *gorm.DB {
	return r.db.
		Preload("Project").Preload("Assignee").Preload("Labels").Preload("Links").
		Preload("Parent.Project").
		Preload("Subtasks.Project").
		Preload("DependsOn.Project").
		Preload("Blocks.Project")
}

// List returns issues matching the filters, newest first.
func (r *Repository) List(f ListFilters) ([]Issue, error) {
	q := r.withRelations().
		Joins("JOIN projects ON projects.id = issues.project_id").
		Order("issues.created_at DESC")

	if f.Project != "" {
		q = q.Where("projects.key = ?", strings.ToUpper(f.Project))
	}
	if len(f.Statuses) > 0 {
		q = q.Where("issues.status IN ?", f.Statuses)
	}
	if f.Assignee != "" {
		q = q.Joins("JOIN users ON users.id = issues.assignee_id").
			Where("users.name = ?", f.Assignee)
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		// Match the public key too ("965", "PRO-965", "pro-9") — the
		// projects table is already joined above.
		q = q.Where(
			"issues.title LIKE ? OR issues.description LIKE ? OR (projects.key || '-' || issues.number) LIKE ?",
			like, like, strings.ToUpper(like),
		)
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}

	var issues []Issue
	err := q.Find(&issues).Error
	return issues, err
}

// GetByKey looks up an issue by its public identifier, e.g. "TD-12".
func (r *Repository) GetByKey(key string) (*Issue, error) {
	projectKey, number, err := parseKey(key)
	if err != nil {
		return nil, err
	}

	var issue Issue
	err = r.withRelations().
		Joins("JOIN projects ON projects.id = issues.project_id").
		Where("projects.key = ? AND issues.number = ?", projectKey, number).
		First(&issue).Error
	if err != nil {
		return nil, err
	}
	return &issue, nil
}

// Create inserts an issue inside a transaction. The number comes from the
// project's next_number counter (never reused after deletes), unless an
// explicit number is given — the import path for preserving keys from
// another tracker. The counter always ends up past the highest number used.
func (r *Repository) Create(issue *Issue, explicitNumber int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var next int
		if err := tx.Model(&projects.Project{}).
			Where("id = ?", issue.ProjectID).
			Select("next_number").
			Scan(&next).Error; err != nil {
			return err
		}

		issue.Number = next
		if explicitNumber > 0 {
			issue.Number = explicitNumber
		}

		if err := tx.Create(issue).Error; err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("issue number %d is already taken in this project", issue.Number)
			}
			return err
		}

		if issue.Number >= next {
			return tx.Model(&projects.Project{}).
				Where("id = ?", issue.ProjectID).
				Update("next_number", issue.Number+1).Error
		}
		return nil
	})
}

// Update persists changed columns on an issue.
func (r *Repository) Update(issue *Issue, updates map[string]any) error {
	return r.db.Model(issue).Updates(updates).Error
}

// ReplaceLabels swaps the issue's label set.
func (r *Repository) ReplaceLabels(issue *Issue, labelRows []labels.Label) error {
	return r.db.Model(issue).Association("Labels").Replace(labelRows)
}

// ReplaceDependsOn swaps the set of issues this issue is blocked by.
func (r *Repository) ReplaceDependsOn(issue *Issue, deps []Issue) error {
	return r.db.Model(issue).Association("DependsOn").Replace(deps)
}

// WouldCreateParentCycle reports whether setting parentID as the parent of
// issueID would create a loop in the subtask tree (walks up the chain).
func (r *Repository) WouldCreateParentCycle(issueID, parentID uint) (bool, error) {
	current := parentID
	for current != 0 {
		if current == issueID {
			return true, nil
		}
		var next *uint
		err := r.db.Model(&Issue{}).Where("id = ?", current).
			Select("parent_id").Scan(&next).Error
		if err != nil || next == nil {
			return false, err
		}
		current = *next
	}
	return false, nil
}

// DependsReaches reports whether targetID is reachable from fromID by
// following depends_on edges — used to reject circular dependencies.
func (r *Repository) DependsReaches(fromID, targetID uint) (bool, error) {
	visited := map[uint]bool{}
	queue := []uint{fromID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if id == targetID {
			return true, nil
		}
		if visited[id] {
			continue
		}
		visited[id] = true

		var deps []uint
		err := r.db.Table("issue_relations").
			Where("issue_id = ?", id).
			Pluck("depends_on_id", &deps).Error
		if err != nil {
			return false, err
		}
		queue = append(queue, deps...)
	}
	return false, nil
}

// Delete removes an issue; labels and comments cascade via FKs.
func (r *Repository) Delete(issue *Issue) error {
	return r.db.Delete(issue).Error
}

// NextTask returns the highest-priority unstarted issue for a user —
// the query behind the taskdock_get_next_task MCP tool.
// Issues blocked by an unfinished dependency are skipped.
// A non-empty projectKey restricts the search to that project.
func (r *Repository) NextTask(assigneeName, projectKey string) (*Issue, error) {
	q := r.withRelations().
		Joins("JOIN users ON users.id = issues.assignee_id").
		Where("users.name = ?", assigneeName).
		Where("issues.status IN ?", []string{"backlog", "todo"})

	if projectKey != "" {
		q = q.Joins("JOIN projects ON projects.id = issues.project_id").
			Where("projects.key = ?", strings.ToUpper(projectKey))
	}

	var issue Issue
	err := q.
		Where(`NOT EXISTS (
			SELECT 1 FROM issue_relations ir
			JOIN issues dep ON dep.id = ir.depends_on_id
			WHERE ir.issue_id = issues.id
			  AND dep.status NOT IN ('done', 'canceled'))`).
		Order(priorityOrder + ", issues.created_at ASC").
		First(&issue).Error
	if err != nil {
		return nil, err
	}
	return &issue, nil
}

// priorityOrder sorts urgent → high → medium → low → none in SQL.
// An issue due today or overdue outranks everything regardless of priority.
const priorityOrder = `(CASE WHEN issues.due_date IS NOT NULL
		AND issues.due_date <= date('now') THEN 1 ELSE 0 END) DESC,
	CASE issues.priority
	WHEN 'urgent' THEN 4
	WHEN 'high' THEN 3
	WHEN 'medium' THEN 2
	WHEN 'low' THEN 1
	ELSE 0 END DESC`

// parseKey splits "TD-12" into ("TD", 12).
func parseKey(key string) (string, int, error) {
	idx := strings.LastIndex(key, "-")
	if idx <= 0 {
		return "", 0, fmt.Errorf("invalid issue key: %s", key)
	}
	number, err := strconv.Atoi(key[idx+1:])
	if err != nil {
		return "", 0, fmt.Errorf("invalid issue key: %s", key)
	}
	return strings.ToUpper(key[:idx]), number, nil
}
