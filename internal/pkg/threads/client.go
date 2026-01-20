package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ayteuir/backend/internal/config"
)

const (
	baseGraphURL = "https://graph.threads.net"
	baseAuthURL  = "https://threads.net"
)

type Client struct {
	httpClient *http.Client
	cfg        *config.ThreadsConfig
}

func NewClient(cfg *config.ThreadsConfig) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cfg: cfg,
	}
}

func (c *Client) GetAuthorizationURL(state string) string {
	params := url.Values{
		"client_id":     {c.cfg.AppID},
		"redirect_uri":  {c.cfg.RedirectURI},
		"scope":         {"threads_basic,threads_content_publish,threads_manage_replies"},
		"response_type": {"code"},
		"state":         {state},
	}
	return fmt.Sprintf("%s/oauth/authorize?%s", baseAuthURL, params.Encode())
}

func (c *Client) ExchangeCodeForToken(ctx context.Context, code string) (*OAuthResponse, error) {
	data := url.Values{
		"client_id":     {c.cfg.AppID},
		"client_secret": {c.cfg.AppSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {c.cfg.RedirectURI},
		"code":          {code},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseGraphURL+"/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("threads API error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("failed to exchange code: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp OAuthResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (c *Client) ExchangeForLongLivedToken(ctx context.Context, shortLivedToken string) (*LongLivedTokenResponse, error) {
	params := url.Values{
		"grant_type":         {"th_exchange_token"},
		"client_secret":      {c.cfg.AppSecret},
		"access_token":       {shortLivedToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/access_token?%s", baseGraphURL, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get long-lived token: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp LongLivedTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (c *Client) RefreshToken(ctx context.Context, token string) (*RefreshTokenResponse, error) {
	params := url.Values{
		"grant_type":   {"th_refresh_token"},
		"access_token": {token},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/refresh_access_token?%s", baseGraphURL, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to refresh token: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp RefreshTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (c *Client) GetUserProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	params := url.Values{
		"fields":       {"id,username,name,threads_profile_picture_url,threads_biography"},
		"access_token": {accessToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/me?%s", baseGraphURL, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user profile: status %d, body: %s", resp.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

func (c *Client) GetMediaObject(ctx context.Context, accessToken, mediaID string) (*MediaObject, error) {
	params := url.Values{
		"fields":       {"id,media_type,media_url,permalink,username,text,timestamp,shortcode,is_quote_post,owner"},
		"access_token": {accessToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s?%s", baseGraphURL, mediaID, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get media object: status %d, body: %s", resp.StatusCode, string(body))
	}

	var media MediaObject
	if err := json.Unmarshal(body, &media); err != nil {
		return nil, err
	}

	return &media, nil
}

func (c *Client) CreateReply(ctx context.Context, accessToken, userID, text, replyToID string) (string, error) {
	params := url.Values{
		"media_type":   {"TEXT"},
		"text":         {text},
		"reply_to_id":  {replyToID},
		"access_token": {accessToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/%s/threads?%s", baseGraphURL, userID, params.Encode()), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create reply container: status %d, body: %s", resp.StatusCode, string(body))
	}

	var containerResp CreateMediaContainerResponse
	if err := json.Unmarshal(body, &containerResp); err != nil {
		return "", err
	}

	return c.publishMedia(ctx, accessToken, userID, containerResp.ID)
}

func (c *Client) publishMedia(ctx context.Context, accessToken, userID, creationID string) (string, error) {
	params := url.Values{
		"creation_id":  {creationID},
		"access_token": {accessToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/%s/threads_publish?%s", baseGraphURL, userID, params.Encode()), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to publish reply: status %d, body: %s", resp.StatusCode, string(body))
	}

	var publishResp PublishMediaResponse
	if err := json.Unmarshal(body, &publishResp); err != nil {
		return "", err
	}

	return publishResp.ID, nil
}

// GetReplies fetches replies to a specific media post
func (c *Client) GetReplies(ctx context.Context, accessToken, mediaID string, reverse bool) (*RepliesResponse, error) {
	params := url.Values{
		"fields":       {"id,text,timestamp,media_type,permalink,username,is_reply,root_post,replied_to,hide_status"},
		"access_token": {accessToken},
	}
	if reverse {
		params.Set("reverse", "true")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s/replies?%s", baseGraphURL, mediaID, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("threads API error: %s (code: %d)", errResp.Error.Message, errResp.Error.Code)
		}
		return nil, fmt.Errorf("failed to get replies: status %d, body: %s", resp.StatusCode, string(body))
	}

	var repliesResp RepliesResponse
	if err := json.Unmarshal(body, &repliesResp); err != nil {
		return nil, err
	}

	return &repliesResp, nil
}

// GetConversation fetches the conversation (all replies in a thread hierarchy)
func (c *Client) GetConversation(ctx context.Context, accessToken, mediaID string, reverse bool) (*RepliesResponse, error) {
	params := url.Values{
		"fields":       {"id,text,timestamp,media_type,permalink,username,is_reply,root_post,replied_to,hide_status"},
		"access_token": {accessToken},
	}
	if reverse {
		params.Set("reverse", "true")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s/conversation?%s", baseGraphURL, mediaID, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("threads API error: %s (code: %d)", errResp.Error.Message, errResp.Error.Code)
		}
		return nil, fmt.Errorf("failed to get conversation: status %d, body: %s", resp.StatusCode, string(body))
	}

	var repliesResp RepliesResponse
	if err := json.Unmarshal(body, &repliesResp); err != nil {
		return nil, err
	}

	return &repliesResp, nil
}

// GetUserThreads fetches the user's threads posts
func (c *Client) GetUserThreads(ctx context.Context, accessToken, userID string, limit int, since *time.Time) (*ConversationsResponse, error) {
	params := url.Values{
		"fields":       {"id,text,timestamp,media_type,permalink,username"},
		"access_token": {accessToken},
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if since != nil {
		params.Set("since", fmt.Sprintf("%d", since.Unix()))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s/threads?%s", baseGraphURL, userID, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("threads API error: %s (code: %d)", errResp.Error.Message, errResp.Error.Code)
		}
		return nil, fmt.Errorf("failed to get user threads: status %d, body: %s", resp.StatusCode, string(body))
	}

	var threadsResp ConversationsResponse
	if err := json.Unmarshal(body, &threadsResp); err != nil {
		return nil, err
	}

	return &threadsResp, nil
}
