package domain

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Template struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID   `bson:"user_id" json:"user_id"`
	Name        string               `bson:"name" json:"name"`
	MentionType MentionType          `bson:"mention_type" json:"mention_type"`
	Content     string               `bson:"content" json:"content"`
	Variables   []string             `bson:"variables" json:"variables"`
	IsActive    bool                 `bson:"is_active" json:"is_active"`
	Priority    int                  `bson:"priority" json:"priority"`
	Conditions  *TemplateConditions  `bson:"conditions,omitempty" json:"conditions,omitempty"`
	CreatedAt   time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time            `bson:"updated_at" json:"updated_at"`
}

type TemplateConditions struct {
	Keywords           []string `bson:"keywords,omitempty" json:"keywords,omitempty"`
	SentimentThreshold *float64 `bson:"sentiment_threshold,omitempty" json:"sentiment_threshold,omitempty"`
}

type TemplateVariables struct {
	Username     string
	DisplayName  string
	Content      string
	MentionType  string
	Sentiment    string
}

func NewTemplate(userID primitive.ObjectID, name string, mentionType MentionType, content string) *Template {
	now := time.Now()
	return &Template{
		UserID:      userID,
		Name:        name,
		MentionType: mentionType,
		Content:     content,
		Variables:   extractVariables(content),
		IsActive:    true,
		Priority:    10,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (t *Template) Render(vars TemplateVariables) (string, error) {
	tmpl, err := template.New("reply").Parse(t.Content)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (t *Template) MatchesConditions(analysis *MentionAnalysis) bool {
	if t.Conditions == nil {
		return true
	}

	if t.Conditions.SentimentThreshold != nil {
		if analysis.Sentiment > *t.Conditions.SentimentThreshold {
			return false
		}
	}

	if len(t.Conditions.Keywords) > 0 {
		found := false
		contentLower := strings.ToLower(analysis.RawAnalysis)
		for _, keyword := range t.Conditions.Keywords {
			if strings.Contains(contentLower, strings.ToLower(keyword)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (t *Template) Update(name string, content string, isActive bool, priority int) {
	t.Name = name
	t.Content = content
	t.Variables = extractVariables(content)
	t.IsActive = isActive
	t.Priority = priority
	t.UpdatedAt = time.Now()
}

var variableRegex = regexp.MustCompile(`\{\{\.(\w+)\}\}`)

func extractVariables(content string) []string {
	matches := variableRegex.FindAllStringSubmatch(content, -1)
	vars := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			vars = append(vars, match[1])
			seen[match[1]] = true
		}
	}

	return vars
}
