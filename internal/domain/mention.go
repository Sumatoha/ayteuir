package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MentionType string

const (
	MentionTypeComplaint MentionType = "complaint"
	MentionTypePositive  MentionType = "positive"
	MentionTypeQuestion  MentionType = "question"
	MentionTypeNeutral   MentionType = "neutral"
	MentionTypeSpam      MentionType = "spam"
)

type MentionStatus string

const (
	MentionStatusPending    MentionStatus = "pending"
	MentionStatusProcessing MentionStatus = "processing"
	MentionStatusReplied    MentionStatus = "replied"
	MentionStatusSkipped    MentionStatus = "skipped"
	MentionStatusFailed     MentionStatus = "failed"
)

type Mention struct {
	ID                primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID            primitive.ObjectID  `bson:"user_id" json:"user_id"`
	ThreadsPostID     string              `bson:"threads_post_id" json:"threads_post_id"`
	ThreadsParentID   string              `bson:"threads_parent_id,omitempty" json:"threads_parent_id,omitempty"`
	Author            MentionAuthor       `bson:"author" json:"author"`
	Content           string              `bson:"content" json:"content"`
	MediaURLs         []string            `bson:"media_urls" json:"media_urls"`
	Analysis          *MentionAnalysis    `bson:"analysis,omitempty" json:"analysis,omitempty"`
	Status            MentionStatus       `bson:"status" json:"status"`
	SkipReason        string              `bson:"skip_reason,omitempty" json:"skip_reason,omitempty"`
	ReplyID           *primitive.ObjectID `bson:"reply_id,omitempty" json:"reply_id,omitempty"`
	WebhookReceivedAt time.Time           `bson:"webhook_received_at" json:"webhook_received_at"`
	ProcessedAt       *time.Time          `bson:"processed_at,omitempty" json:"processed_at,omitempty"`
	CreatedAt         time.Time           `bson:"created_at" json:"created_at"`
}

type MentionAuthor struct {
	ThreadsUserID string `bson:"threads_user_id" json:"threads_user_id"`
	Username      string `bson:"username" json:"username"`
	DisplayName   string `bson:"display_name" json:"display_name"`
}

type MentionAnalysis struct {
	MentionType   MentionType `bson:"mention_type" json:"mention_type"`
	Sentiment     float64     `bson:"sentiment" json:"sentiment"`
	Intent        string      `bson:"intent" json:"intent"`
	Urgency       string      `bson:"urgency" json:"urgency"`
	Keywords      []string    `bson:"keywords" json:"keywords"`
	SuggestedTone string      `bson:"suggested_tone" json:"suggested_tone"`
	RawAnalysis   string      `bson:"raw_analysis,omitempty" json:"-"`
}

func NewMention(userID primitive.ObjectID, threadsPostID string, author MentionAuthor, content string) *Mention {
	now := time.Now()
	return &Mention{
		UserID:            userID,
		ThreadsPostID:     threadsPostID,
		Author:            author,
		Content:           content,
		MediaURLs:         []string{},
		Status:            MentionStatusPending,
		WebhookReceivedAt: now,
		CreatedAt:         now,
	}
}

func (m *Mention) SetAnalysis(analysis *MentionAnalysis) {
	m.Analysis = analysis
}

func (m *Mention) MarkProcessing() {
	m.Status = MentionStatusProcessing
}

func (m *Mention) MarkReplied(replyID primitive.ObjectID) {
	m.Status = MentionStatusReplied
	m.ReplyID = &replyID
	now := time.Now()
	m.ProcessedAt = &now
}

func (m *Mention) MarkSkipped(reason string) {
	m.Status = MentionStatusSkipped
	m.SkipReason = reason
	now := time.Now()
	m.ProcessedAt = &now
}

func (m *Mention) MarkFailed(reason string) {
	m.Status = MentionStatusFailed
	m.SkipReason = reason
	now := time.Now()
	m.ProcessedAt = &now
}
