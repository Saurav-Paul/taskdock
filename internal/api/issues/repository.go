package issues

// Data access layer — all database queries for issues.

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/Saurav-Paul/taskdock/internal/api/labels"
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
func (r *Repository) withRelations() *gorm.DB {
	return r.db.Preload("Project").Preload("Assignee").Preload("Labels")
}

// List returns issues matching the filters, newest first.
func (r *Repository) List(f ListFilters) ([]Issue, error) {
	q := r.withRelations().
		Joins("JOIN projects ON projects.id = issues.project_id").
		Order("issues.created_at DESC")

	if f.Project != "" {
		q = q.Where("projects.key = ?", strings.ToUpper(f.Project))
	}
	if f.Status != "" {
		q = q.Where("issues.status = ?", f.Status)
	}
	if f.Assignee != "" {
		q = q.Joins("JOIN users ON users.id = issues.assignee_id").
			Where("users.name = ?", f.Assignee)
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		q = q.Where("issues.title LIKE ? OR issues.description LIKE ?", like, like)
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

// Create inserts an issue, assigning the next per-project number inside a
// transaction so concurrent creates can't collide.
func (r *Repository) Create(issue *Issue) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var max sql.NullInt64
		if err := tx.Model(&Issue{}).
			Where("project_id = ?", issue.ProjectID).
			Select("MAX(number)").
			Scan(&max).Error; err != nil {
			return err
		}
		issue.Number = int(max.Int64) + 1
		return tx.Create(issue).Error
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

// Delete removes an issue; labels and comments cascade via FKs.
func (r *Repository) Delete(issue *Issue) error {
	return r.db.Delete(issue).Error
}

// NextTask returns the highest-priority unstarted issue for a user —
// the query behind the taskdock_get_next_task MCP tool.
func (r *Repository) NextTask(assigneeName string) (*Issue, error) {
	var issue Issue
	err := r.withRelations().
		Joins("JOIN users ON users.id = issues.assignee_id").
		Where("users.name = ?", assigneeName).
		Where("issues.status IN ?", []string{"backlog", "todo"}).
		Order(priorityOrder + ", issues.created_at ASC").
		First(&issue).Error
	if err != nil {
		return nil, err
	}
	return &issue, nil
}

// priorityOrder sorts urgent → high → medium → low → none in SQL.
const priorityOrder = `CASE issues.priority
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
