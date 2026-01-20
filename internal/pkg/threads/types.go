package threads

import "time"

type OAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type LongLivedTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type UserProfile struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	Name              string `json:"name"`
	ThreadsProfileURL string `json:"threads_profile_picture_url"`
	Biography         string `json:"threads_biography"`
}

type WebhookPayload struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

type WebhookEntry struct {
	ID      string          `json:"id"`
	Time    int64           `json:"time"`
	Changes []WebhookChange `json:"changes"`
}

type WebhookChange struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

type MentionValue struct {
	From struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"from"`
	MediaID   string `json:"media_id"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

type MediaObject struct {
	ID           string    `json:"id"`
	MediaType    string    `json:"media_type"`
	MediaURL     string    `json:"media_url"`
	Permalink    string    `json:"permalink"`
	Username     string    `json:"username"`
	Text         string    `json:"text"`
	Timestamp    time.Time `json:"timestamp"`
	ShortCode    string    `json:"shortcode"`
	IsQuotePost  bool      `json:"is_quote_post"`
	Owner        MediaOwner `json:"owner"`
}

type MediaOwner struct {
	ID string `json:"id"`
}

type CreateMediaContainerRequest struct {
	MediaType  string `json:"media_type"`
	Text       string `json:"text"`
	ReplyToID  string `json:"reply_to_id,omitempty"`
}

type CreateMediaContainerResponse struct {
	ID string `json:"id"`
}

type PublishMediaRequest struct {
	CreationID string `json:"creation_id"`
}

type PublishMediaResponse struct {
	ID string `json:"id"`
}

type ErrorResponse struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
}

// Conversations/Replies API types

type ConversationsResponse struct {
	Data   []ConversationThread `json:"data"`
	Paging *Paging              `json:"paging,omitempty"`
}

type ConversationThread struct {
	ID        string `json:"id"`
	Text      string `json:"text,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Permalink string `json:"permalink,omitempty"`
	Username  string `json:"username,omitempty"`
}

type RepliesResponse struct {
	Data   []ReplyThread `json:"data"`
	Paging *Paging       `json:"paging,omitempty"`
}

type ReplyThread struct {
	ID            string   `json:"id"`
	Text          string   `json:"text"`
	Timestamp     string   `json:"timestamp"`
	MediaType     string   `json:"media_type"`
	Permalink     string   `json:"permalink"`
	Username      string   `json:"username"`
	IsReply       bool     `json:"is_reply"`
	RootPost      *PostRef `json:"root_post,omitempty"`
	RepliedTo     *PostRef `json:"replied_to,omitempty"`
	HideStatus    string   `json:"hide_status,omitempty"`
	ReplyAudience string   `json:"reply_audience,omitempty"`
}

type PostRef struct {
	ID string `json:"id"`
}

type Paging struct {
	Cursors *Cursors `json:"cursors,omitempty"`
	Next    string   `json:"next,omitempty"`
	Previous string  `json:"previous,omitempty"`
}

type Cursors struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}
