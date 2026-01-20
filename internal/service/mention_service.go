package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ayteuir/backend/internal/domain"
	"github.com/ayteuir/backend/internal/pkg/logger"
	openaiPkg "github.com/ayteuir/backend/internal/pkg/openai"
	"github.com/ayteuir/backend/internal/pkg/threads"
	"github.com/ayteuir/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MentionService struct {
	mentionRepo   repository.MentionRepository
	templateRepo  repository.TemplateRepository
	replyRepo     repository.ReplyRepository
	userRepo      repository.UserRepository
	threadsClient *threads.Client
	openaiClient  *openaiPkg.Client
	authService   *AuthService
}

func NewMentionService(
	mentionRepo repository.MentionRepository,
	templateRepo repository.TemplateRepository,
	replyRepo repository.ReplyRepository,
	userRepo repository.UserRepository,
	threadsClient *threads.Client,
	openaiClient *openaiPkg.Client,
	authService *AuthService,
) *MentionService {
	return &MentionService{
		mentionRepo:   mentionRepo,
		templateRepo:  templateRepo,
		replyRepo:     replyRepo,
		userRepo:      userRepo,
		threadsClient: threadsClient,
		openaiClient:  openaiClient,
		authService:   authService,
	}
}

func (s *MentionService) ProcessMention(ctx context.Context, userID primitive.ObjectID, threadsPostID string, author domain.MentionAuthor, content string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if !user.AutoReplyEnabled {
		logger.Info().Str("user_id", userID.Hex()).Msg("Auto-reply disabled, skipping")
		return nil
	}

	existing, err := s.mentionRepo.GetByThreadsPostID(ctx, threadsPostID)
	if err == nil && existing != nil {
		logger.Info().Str("threads_post_id", threadsPostID).Msg("Mention already exists, skipping")
		return nil
	}

	mention := domain.NewMention(userID, threadsPostID, author, content)
	if err := s.mentionRepo.Create(ctx, mention); err != nil {
		return fmt.Errorf("failed to create mention: %w", err)
	}

	if s.shouldSkipMention(user, author, content) {
		mention.MarkSkipped("matched skip criteria")
		return s.mentionRepo.Update(ctx, mention)
	}

	repliesLastHour, err := s.mentionRepo.CountByUserIDLastHour(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to count replies")
	} else if int(repliesLastHour) >= user.Settings.MaxRepliesPerHour {
		mention.MarkSkipped("rate limit exceeded")
		return s.mentionRepo.Update(ctx, mention)
	}

	go s.processMentionAsync(context.Background(), mention, user)

	return nil
}

func (s *MentionService) processMentionAsync(ctx context.Context, mention *domain.Mention, user *domain.User) {
	mention.MarkProcessing()
	if err := s.mentionRepo.Update(ctx, mention); err != nil {
		logger.Error().Err(err).Str("mention_id", mention.ID.Hex()).Msg("Failed to update mention status")
		return
	}

	analysis, err := s.openaiClient.AnalyzeMention(ctx, mention.Content, mention.Author.Username)
	if err != nil {
		logger.Error().Err(err).Str("mention_id", mention.ID.Hex()).Msg("Failed to analyze mention")
		mention.MarkFailed("AI analysis failed: " + err.Error())
		s.mentionRepo.Update(ctx, mention)
		return
	}

	mention.SetAnalysis(analysis)
	if err := s.mentionRepo.Update(ctx, mention); err != nil {
		logger.Error().Err(err).Msg("Failed to save analysis")
	}

	if analysis.MentionType == domain.MentionTypeSpam {
		mention.MarkSkipped("detected as spam")
		s.mentionRepo.Update(ctx, mention)
		return
	}

	if user.Settings.ReplyDelaySeconds > 0 {
		time.Sleep(time.Duration(user.Settings.ReplyDelaySeconds) * time.Second)
	}

	replyContent, templateID, err := s.generateReply(ctx, user.ID, mention, analysis)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate reply")
		mention.MarkFailed("reply generation failed: " + err.Error())
		s.mentionRepo.Update(ctx, mention)
		return
	}

	reply := domain.NewReply(user.ID, mention.ID, templateID, replyContent)
	if err := s.replyRepo.Create(ctx, reply); err != nil {
		logger.Error().Err(err).Msg("Failed to create reply record")
		return
	}

	accessToken, err := s.authService.GetDecryptedAccessToken(ctx, user.ID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get access token")
		reply.MarkFailed("token error: " + err.Error())
		s.replyRepo.Update(ctx, reply)
		mention.MarkFailed("token error")
		s.mentionRepo.Update(ctx, mention)
		return
	}

	threadsReplyID, err := s.threadsClient.CreateReply(ctx, accessToken, user.ThreadsUserID, replyContent, mention.ThreadsPostID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to post reply to Threads")
		reply.MarkFailed("Threads API error: " + err.Error())
		s.replyRepo.Update(ctx, reply)
		mention.MarkFailed("failed to post reply")
		s.mentionRepo.Update(ctx, mention)
		return
	}

	reply.MarkSent(threadsReplyID, nil)
	s.replyRepo.Update(ctx, reply)

	mention.MarkReplied(reply.ID)
	s.mentionRepo.Update(ctx, mention)

	logger.Info().
		Str("mention_id", mention.ID.Hex()).
		Str("reply_id", reply.ID.Hex()).
		Str("threads_reply_id", threadsReplyID).
		Msg("Successfully replied to mention")
}

func (s *MentionService) shouldSkipMention(user *domain.User, author domain.MentionAuthor, content string) bool {
	contentLower := strings.ToLower(content)
	for _, keyword := range user.Settings.IgnoreKeywords {
		if strings.Contains(contentLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (s *MentionService) generateReply(ctx context.Context, userID primitive.ObjectID, mention *domain.Mention, analysis *domain.MentionAnalysis) (string, *primitive.ObjectID, error) {
	templates, err := s.templateRepo.GetActiveByUserIDAndMentionType(ctx, userID, analysis.MentionType)
	if err != nil {
		return "", nil, err
	}

	var selectedTemplate *domain.Template
	for _, t := range templates {
		if t.MatchesConditions(analysis) {
			selectedTemplate = t
			break
		}
	}

	if selectedTemplate != nil {
		vars := domain.TemplateVariables{
			Username:    mention.Author.Username,
			DisplayName: mention.Author.DisplayName,
			Content:     mention.Content,
			MentionType: string(analysis.MentionType),
			Sentiment:   fmt.Sprintf("%.2f", analysis.Sentiment),
		}

		rendered, err := selectedTemplate.Render(vars)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to render template, using AI generation")
		} else {
			return rendered, &selectedTemplate.ID, nil
		}
	}

	reply, err := s.openaiClient.GenerateReply(ctx, mention.Content, mention.Author.Username, analysis, "")
	if err != nil {
		return "", nil, err
	}

	return reply, nil, nil
}

func (s *MentionService) GetMentions(ctx context.Context, userID primitive.ObjectID, limit, offset int) ([]*domain.Mention, error) {
	return s.mentionRepo.GetByUserID(ctx, userID, limit, offset)
}

func (s *MentionService) GetMention(ctx context.Context, userID, mentionID primitive.ObjectID) (*domain.Mention, error) {
	mention, err := s.mentionRepo.GetByID(ctx, mentionID)
	if err != nil {
		return nil, err
	}

	if mention.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return mention, nil
}

func (s *MentionService) RetryMention(ctx context.Context, userID, mentionID primitive.ObjectID) error {
	mention, err := s.GetMention(ctx, userID, mentionID)
	if err != nil {
		return err
	}

	if mention.Status != domain.MentionStatusFailed {
		return fmt.Errorf("can only retry failed mentions")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	go s.processMentionAsync(context.Background(), mention, user)

	return nil
}

// PullMentionsResult contains results of the pull operation
type PullMentionsResult struct {
	ThreadsChecked int `json:"threads_checked"`
	NewMentions    int `json:"new_mentions"`
	Skipped        int `json:"skipped"`
	Errors         int `json:"errors"`
}

// PullMentions manually fetches mentions/replies from Threads API
// This is a fallback when webhooks are not working
func (s *MentionService) PullMentions(ctx context.Context, userID primitive.ObjectID) (*PullMentionsResult, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	accessToken, err := s.authService.GetDecryptedAccessToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	result := &PullMentionsResult{}

	// Get user's recent threads (last 24 hours or 25 posts)
	since := time.Now().Add(-24 * time.Hour)
	userThreads, err := s.threadsClient.GetUserThreads(ctx, accessToken, user.ThreadsUserID, 25, &since)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID.Hex()).Msg("Failed to fetch user threads")
		return nil, fmt.Errorf("failed to fetch threads: %w", err)
	}

	logger.Info().
		Str("user_id", userID.Hex()).
		Int("threads_count", len(userThreads.Data)).
		Msg("Fetched user threads for pull")

	// For each thread, get replies
	for _, thread := range userThreads.Data {
		result.ThreadsChecked++

		replies, err := s.threadsClient.GetReplies(ctx, accessToken, thread.ID, true)
		if err != nil {
			logger.Warn().Err(err).Str("thread_id", thread.ID).Msg("Failed to get replies for thread")
			result.Errors++
			continue
		}

		// Process each reply as a potential mention
		for _, reply := range replies.Data {
			// Skip own replies
			if reply.Username == user.Username {
				result.Skipped++
				continue
			}

			// Check if we already have this mention
			existing, err := s.mentionRepo.GetByThreadsPostID(ctx, reply.ID)
			if err == nil && existing != nil {
				result.Skipped++
				continue
			}

			// Create new mention
			author := domain.MentionAuthor{
				ThreadsUserID: reply.Username, // We don't have the actual ID from this endpoint
				Username:      reply.Username,
			}

			if err := s.ProcessMention(ctx, userID, reply.ID, author, reply.Text); err != nil {
				logger.Error().Err(err).Str("reply_id", reply.ID).Msg("Failed to process pulled mention")
				result.Errors++
				continue
			}

			result.NewMentions++
			logger.Info().
				Str("reply_id", reply.ID).
				Str("username", reply.Username).
				Msg("Processed new mention from pull")
		}
	}

	logger.Info().
		Str("user_id", userID.Hex()).
		Int("threads_checked", result.ThreadsChecked).
		Int("new_mentions", result.NewMentions).
		Int("skipped", result.Skipped).
		Int("errors", result.Errors).
		Msg("Pull mentions completed")

	return result, nil
}
