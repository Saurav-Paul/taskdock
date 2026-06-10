package issues

// URL attachments on issues — PR links, docs, dashboards. Lives inside the
// issues package because links are purely issue-scoped and appear in every
// issue response.

import (
	"fmt"
	"strings"
	"time"
)

// Link is the GORM model for the "links" table.
type Link struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IssueID   uint      `gorm:"not null" json:"-"`
	URL       string    `gorm:"not null" json:"url"`
	Title     string    `gorm:"not null;default:''" json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

func (Link) TableName() string { return "links" }

// LinkCreate is the JSON body for POST /api/issues/:key/links.
type LinkCreate struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// AddLink attaches a URL to an issue.
func (s *Service) AddLink(issueKey string, req LinkCreate) (*Link, error) {
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		return nil, fmt.Errorf("'url' must start with http:// or https://")
	}

	issue, err := s.repo.GetByKey(issueKey)
	if err != nil {
		return nil, fmt.Errorf("issue not found: %s", issueKey)
	}

	link := Link{IssueID: issue.ID, URL: req.URL, Title: req.Title}
	if err := s.repo.db.Create(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

// DeleteLink removes a link from an issue.
func (s *Service) DeleteLink(issueKey string, linkID uint) error {
	issue, err := s.repo.GetByKey(issueKey)
	if err != nil {
		return fmt.Errorf("issue not found: %s", issueKey)
	}

	result := s.repo.db.Where("id = ? AND issue_id = ?", linkID, issue.ID).Delete(&Link{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("link %d not found on %s", linkID, issueKey)
	}
	return nil
}
