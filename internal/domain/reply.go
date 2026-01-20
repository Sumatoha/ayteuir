package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReplyStatus string

const (
	ReplyStatusPending ReplyStatus = "pending"
	ReplyStatusSent    ReplyStatus = "sent"
	ReplyStatusFailed  ReplyStatus = "failed"
)

type Reply struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID          primitive.ObjectID  `bson:"user_id" json:"user_id"`
	MentionID       primitive.ObjectID  `bson:"mention_id" json:"mention_id"`
	TemplateID      *primitive.ObjectID `bson:"template_id,omitempty" json:"template_id,omitempty"`
	ThreadsReplyID  string              `bson:"threads_reply_id,omitempty" json:"threads_reply_id,omitempty"`
	Content         string              `bson:"content" json:"content"`
	Status          ReplyStatus         `bson:"status" json:"status"`
	Error           string              `bson:"error,omitempty" json:"error,omitempty"`
	ThreadsResponse map[string]any      `bson:"threads_response,omitempty" json:"-"`
	SentAt          *time.Time          `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
	CreatedAt       time.Time           `bson:"created_at" json:"created_at"`
}

func NewReply(userID, mentionID primitive.ObjectID, templateID *primitive.ObjectID, content string) *Reply {
	return &Reply{
		UserID:     userID,
		MentionID:  mentionID,
		TemplateID: templateID,
		Content:    content,
		Status:     ReplyStatusPending,
		CreatedAt:  time.Now(),
	}
}

func (r *Reply) MarkSent(threadsReplyID string, response map[string]any) {
	r.Status = ReplyStatusSent
	r.ThreadsReplyID = threadsReplyID
	r.ThreadsResponse = response
	now := time.Now()
	r.SentAt = &now
}

func (r *Reply) MarkFailed(err string) {
	r.Status = ReplyStatusFailed
	r.Error = err
}
