package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ayteuir/backend/internal/config"
	"github.com/ayteuir/backend/internal/domain"
	"github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	cfg    *config.OpenAIConfig
}

func NewClient(cfg *config.OpenAIConfig) *Client {
	return &Client{
		client: openai.NewClient(cfg.APIKey),
		cfg:    cfg,
	}
}

type AnalysisResult struct {
	MentionType   string   `json:"mention_type"`
	Sentiment     float64  `json:"sentiment"`
	Intent        string   `json:"intent"`
	Urgency       string   `json:"urgency"`
	Keywords      []string `json:"keywords"`
	SuggestedTone string   `json:"suggested_tone"`
}

func (c *Client) AnalyzeMention(ctx context.Context, mentionText, authorUsername string) (*domain.MentionAnalysis, error) {
	systemPrompt := `You are an AI assistant that analyzes social media mentions for a business account.
Your task is to classify mentions and determine the appropriate response strategy.

You must respond with a valid JSON object containing exactly these fields:
- mention_type: one of "complaint", "positive", "question", "neutral", "spam"
- sentiment: a number from -1.0 (very negative) to 1.0 (very positive)
- intent: brief description of what the user wants (e.g., "seeking_resolution", "giving_praise", "asking_question", "general_comment")
- urgency: one of "high", "medium", "low"
- keywords: array of 1-5 key words/phrases from the mention
- suggested_tone: recommended tone for reply (e.g., "apologetic", "grateful", "helpful", "friendly")

Classification guidelines:
- complaint: negative feedback, issues, problems, frustration
- positive: praise, compliments, thanks, recommendations
- question: seeking information, how-to, availability inquiries
- neutral: general mentions without strong sentiment
- spam: promotional content, bots, irrelevant mentions`

	userPrompt := fmt.Sprintf(`Analyze this social media mention:

Author: @%s
Content: "%s"

Provide your analysis as a JSON object.`, authorUsername, mentionText)

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.cfg.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		MaxTokens: c.cfg.MaxTokens,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	content := resp.Choices[0].Message.Content

	var result AnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	return &domain.MentionAnalysis{
		MentionType:   domain.MentionType(result.MentionType),
		Sentiment:     result.Sentiment,
		Intent:        result.Intent,
		Urgency:       result.Urgency,
		Keywords:      result.Keywords,
		SuggestedTone: result.SuggestedTone,
		RawAnalysis:   content,
	}, nil
}

func (c *Client) GenerateReply(ctx context.Context, mentionText, authorUsername string, analysis *domain.MentionAnalysis, templateHint string) (string, error) {
	systemPrompt := `You are a helpful social media manager. Generate a brief, professional reply to a mention.
Keep the reply concise (under 280 characters), friendly, and appropriate for the context.
Do not use hashtags unless specifically relevant. Sign off naturally without formal signatures.`

	userPrompt := fmt.Sprintf(`Generate a reply to this mention:

Author: @%s
Content: "%s"

Analysis:
- Type: %s
- Sentiment: %.2f
- Suggested tone: %s

%s

Generate a single reply message.`,
		authorUsername, mentionText,
		analysis.MentionType, analysis.Sentiment, analysis.SuggestedTone,
		templateHint)

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.cfg.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		MaxTokens: 150,
	})
	if err != nil {
		return "", fmt.Errorf("openai API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
