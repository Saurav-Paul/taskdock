package issues

// Business logic for issues — validation, name→row resolution, key generation.

import (
	"fmt"
	"time"

	"github.com/Saurav-Paul/taskdock/internal/api/labels"
	"github.com/Saurav-Paul/taskdock/internal/api/projects"
	"github.com/Saurav-Paul/taskdock/internal/api/users"
)

// Service handles business logic for issues. It resolves project keys,
// assignee handles, and label names through the sibling domain services.
type Service struct {
	repo     *Repository
	projects *projects.Service
	users    *users.Service
	labels   *labels.Service
}

// NewService creates a new Service with its domain dependencies.
func NewService(repo *Repository, p *projects.Service, u *users.Service, l *labels.Service) *Service {
	return &Service{repo: repo, projects: p, users: u, labels: l}
}

// List returns issues matching the filters as API responses.
func (s *Service) List(f ListFilters) ([]IssueResponse, error) {
	rows, err := s.repo.List(f)
	if err != nil {
		return nil, err
	}
	out := make([]IssueResponse, 0, len(rows))
	for i := range rows {
		out = append(out, ToResponse(&rows[i]))
	}
	return out, nil
}

// Get returns a single issue by key, e.g. "TD-12".
func (s *Service) Get(key string) (*IssueResponse, error) {
	issue, err := s.repo.GetByKey(key)
	if err != nil {
		return nil, err
	}
	resp := ToResponse(issue)
	return &resp, nil
}

// GetModel returns the raw issue row — used by the comments domain to
// resolve an issue key to its id.
func (s *Service) GetModel(key string) (*Issue, error) {
	return s.repo.GetByKey(key)
}

// Create validates the request, assigns the next per-project number, and
// returns the created issue.
func (s *Service) Create(req IssueCreate) (*IssueResponse, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("'title' is required")
	}

	project, err := s.projects.GetByKey(req.Project)
	if err != nil {
		return nil, fmt.Errorf("unknown project: %s", req.Project)
	}

	status := req.Status
	if status == "" {
		status = "todo"
	}
	if !IsValidStatus(status) {
		return nil, fmt.Errorf("invalid status: %s (valid: %v)", status, ValidStatuses)
	}

	priority := req.Priority
	if priority == "" {
		priority = "none"
	}
	if !IsValidPriority(priority) {
		return nil, fmt.Errorf("invalid priority: %s (valid: %v)", priority, ValidPriorities)
	}

	issue := Issue{
		ProjectID:   project.ID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
	}

	// Issues created directly in a working/terminal state get stamped too
	// (e.g. importing already-done issues from another tracker).
	now := time.Now().UTC()
	switch status {
	case "in_progress", "in_review":
		issue.StartedAt = &now
	case "done", "canceled":
		issue.CompletedAt = &now
	}

	if req.Assignee != "" {
		user, err := s.users.GetByName(req.Assignee)
		if err != nil {
			return nil, fmt.Errorf("unknown user: %s", req.Assignee)
		}
		issue.AssigneeID = &user.ID
	}

	if req.Parent != "" {
		parent, err := s.repo.GetByKey(req.Parent)
		if err != nil {
			return nil, fmt.Errorf("parent issue not found: %s", req.Parent)
		}
		issue.ParentID = &parent.ID
	}

	if err := s.repo.Create(&issue, req.Number); err != nil {
		return nil, err
	}

	if len(req.Labels) > 0 {
		labelRows, err := s.labels.GetOrCreate(req.Labels)
		if err != nil {
			return nil, err
		}
		if err := s.repo.ReplaceLabels(&issue, labelRows); err != nil {
			return nil, err
		}
	}

	if len(req.DependsOn) > 0 {
		deps, err := s.resolveDependencies(&issue, req.DependsOn)
		if err != nil {
			return nil, err
		}
		if err := s.repo.ReplaceDependsOn(&issue, deps); err != nil {
			return nil, err
		}
	}

	// Re-fetch with relations so the response includes project/assignee/labels.
	return s.Get(fmt.Sprintf("%s-%d", project.Key, issue.Number))
}

// Update applies non-nil fields to the issue and returns the updated response.
func (s *Service) Update(key string, req IssueUpdate) (*IssueResponse, error) {
	issue, err := s.repo.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("issue not found: %s", key)
	}

	updates := map[string]any{}
	if req.Title != nil {
		if *req.Title == "" {
			return nil, fmt.Errorf("'title' cannot be empty")
		}
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		if !IsValidStatus(*req.Status) {
			return nil, fmt.Errorf("invalid status: %s (valid: %v)", *req.Status, ValidStatuses)
		}
		updates["status"] = *req.Status

		// Cycle-time stamps: first transition into a working state sets
		// started_at, first transition into a terminal state sets
		// completed_at. Neither is ever overwritten.
		now := time.Now().UTC()
		switch *req.Status {
		case "in_progress", "in_review":
			if issue.StartedAt == nil {
				updates["started_at"] = now
			}
		case "done", "canceled":
			if issue.CompletedAt == nil {
				updates["completed_at"] = now
			}
		}
	}
	if req.Priority != nil {
		if !IsValidPriority(*req.Priority) {
			return nil, fmt.Errorf("invalid priority: %s (valid: %v)", *req.Priority, ValidPriorities)
		}
		updates["priority"] = *req.Priority
	}
	if req.Assignee != nil {
		if *req.Assignee == "" {
			updates["assignee_id"] = nil // unassign
		} else {
			user, err := s.users.GetByName(*req.Assignee)
			if err != nil {
				return nil, fmt.Errorf("unknown user: %s", *req.Assignee)
			}
			updates["assignee_id"] = user.ID
		}
	}
	if req.Parent != nil {
		if *req.Parent == "" {
			updates["parent_id"] = nil // detach from parent
		} else {
			parent, err := s.repo.GetByKey(*req.Parent)
			if err != nil {
				return nil, fmt.Errorf("parent issue not found: %s", *req.Parent)
			}
			if parent.ID == issue.ID {
				return nil, fmt.Errorf("an issue cannot be its own parent")
			}
			cycle, err := s.repo.WouldCreateParentCycle(issue.ID, parent.ID)
			if err != nil {
				return nil, err
			}
			if cycle {
				return nil, fmt.Errorf("cannot set %s as parent: it is a subtask of %s", *req.Parent, key)
			}
			updates["parent_id"] = parent.ID
		}
	}

	if len(updates) > 0 {
		if err := s.repo.Update(issue, updates); err != nil {
			return nil, err
		}
	}

	if req.Labels != nil {
		labelRows, err := s.labels.GetOrCreate(*req.Labels)
		if err != nil {
			return nil, err
		}
		if err := s.repo.ReplaceLabels(issue, labelRows); err != nil {
			return nil, err
		}
	}

	if req.DependsOn != nil {
		deps, err := s.resolveDependencies(issue, *req.DependsOn)
		if err != nil {
			return nil, err
		}
		if err := s.repo.ReplaceDependsOn(issue, deps); err != nil {
			return nil, err
		}
	}

	return s.Get(key)
}

// resolveDependencies turns issue keys into rows, rejecting self-references
// and circular dependency chains.
func (s *Service) resolveDependencies(issue *Issue, keys []string) ([]Issue, error) {
	deps := make([]Issue, 0, len(keys))
	for _, depKey := range keys {
		dep, err := s.repo.GetByKey(depKey)
		if err != nil {
			return nil, fmt.Errorf("dependency issue not found: %s", depKey)
		}
		if dep.ID == issue.ID {
			return nil, fmt.Errorf("an issue cannot depend on itself")
		}
		// Reject cycles: the dependency must not (transitively) depend on us.
		cycle, err := s.repo.DependsReaches(dep.ID, issue.ID)
		if err != nil {
			return nil, err
		}
		if cycle {
			return nil, fmt.Errorf("circular dependency: %s already depends on this issue", depKey)
		}
		deps = append(deps, *dep)
	}
	return deps, nil
}

// Delete removes an issue by key.
func (s *Service) Delete(key string) error {
	issue, err := s.repo.GetByKey(key)
	if err != nil {
		return fmt.Errorf("issue not found: %s", key)
	}
	return s.repo.Delete(issue)
}

// NextTask returns the highest-priority unstarted issue assigned to the
// user, optionally restricted to one project.
func (s *Service) NextTask(assigneeName, projectKey string) (*IssueResponse, error) {
	issue, err := s.repo.NextTask(assigneeName, projectKey)
	if err != nil {
		return nil, err
	}
	resp := ToResponse(issue)
	return &resp, nil
}
