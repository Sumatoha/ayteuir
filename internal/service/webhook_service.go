package service

import (
	"context"

	"github.com/ayteuir/backend/internal/domain"
	"github.com/ayteuir/backend/internal/pkg/logger"
	"github.com/ayteuir/backend/internal/pkg/threads"
	"github.com/ayteuir/backend/internal/repository"
)

type WebhookService struct {
	verifier       *threads.WebhookVerifier
	threadsClient  *threads.Client
	userRepo       repository.UserRepository
	mentionService *MentionService
}

func NewWebhookService(
	verifier *threads.WebhookVerifier,
	threadsClient *threads.Client,
	userRepo repository.UserRepository,
	mentionService *MentionService,
) *WebhookService {
	return &WebhookService{
		verifier:       verifier,
		threadsClient:  threadsClient,
		userRepo:       userRepo,
		mentionService: mentionService,
	}
}

func (s *WebhookService) VerifyChallenge(mode, token, challenge string) (string, bool) {
	return s.verifier.VerifyChallenge(mode, token, challenge)
}

func (s *WebhookService) VerifySignature(payload []byte, signature string) bool {
	return s.verifier.VerifySignature(payload, signature)
}

func (s *WebhookService) ProcessWebhook(ctx context.Context, payload []byte) error {
	webhookPayload, err := threads.ParseWebhookPayload(payload)
	if err != nil {
		return err
	}

	mentions := threads.ExtractMentions(webhookPayload)

	for _, mention := range mentions {
		if err := s.processMention(ctx, webhookPayload, mention); err != nil {
			logger.Error().Err(err).Interface("mention", mention).Msg("Failed to process mention")
		}
	}

	return nil
}

func (s *WebhookService) processMention(ctx context.Context, payload *threads.WebhookPayload, mention threads.MentionValue) error {
	if len(payload.Entry) == 0 {
		return nil
	}

	threadsUserID := payload.Entry[0].ID

	user, err := s.userRepo.GetByThreadsUserID(ctx, threadsUserID)
	if err != nil {
		if domain.IsNotFound(err) {
			logger.Warn().Str("threads_user_id", threadsUserID).Msg("User not found for webhook")
			return nil
		}
		return err
	}

	author := domain.MentionAuthor{
		ThreadsUserID: mention.From.ID,
		Username:      mention.From.Username,
		DisplayName:   mention.From.Username,
	}

	return s.mentionService.ProcessMention(ctx, user.ID, mention.MediaID, author, mention.Text)
}
