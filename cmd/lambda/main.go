package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/ayteuir/backend/internal/config"
	"github.com/ayteuir/backend/internal/handler"
	"github.com/ayteuir/backend/internal/middleware"
	"github.com/ayteuir/backend/internal/pkg/logger"
	"github.com/ayteuir/backend/internal/pkg/openai"
	"github.com/ayteuir/backend/internal/pkg/threads"
	"github.com/ayteuir/backend/internal/repository/mongodb"
	"github.com/ayteuir/backend/internal/service"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

var chiLambda *chiadapter.ChiLambda

func init() {
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
		os.Exit(1)
	}

	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info().Str("env", cfg.App.Env).Msg("Initializing Lambda")

	ctx := context.Background()

	mongoClient, err := mongodb.NewClient(&cfg.MongoDB)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to MongoDB")
		os.Exit(1)
	}

	if err := mongoClient.CreateIndexes(ctx); err != nil {
		logger.Warn().Err(err).Msg("Failed to create indexes")
	}

	userRepo := mongodb.NewUserRepository(mongoClient)
	templateRepo := mongodb.NewTemplateRepository(mongoClient)
	mentionRepo := mongodb.NewMentionRepository(mongoClient)
	replyRepo := mongodb.NewReplyRepository(mongoClient)

	threadsClient := threads.NewClient(&cfg.Threads)
	openaiClient := openai.NewClient(&cfg.OpenAI)
	webhookVerifier := threads.NewWebhookVerifier(cfg.Threads.AppSecret, cfg.Threads.WebhookVerifyToken)

	authService := service.NewAuthService(userRepo, threadsClient, cfg)
	userService := service.NewUserService(userRepo)
	templateService := service.NewTemplateService(templateRepo)
	mentionService := service.NewMentionService(
		mentionRepo,
		templateRepo,
		replyRepo,
		userRepo,
		threadsClient,
		openaiClient,
		authService,
	)
	webhookService := service.NewWebhookService(webhookVerifier, threadsClient, userRepo, mentionService)

	healthHandler := handler.NewHealthHandler(mongoClient)
	authHandler := handler.NewAuthHandler(authService, userService, cfg)
	webhookHandler := handler.NewWebhookHandler(webhookService)
	templateHandler := handler.NewTemplateHandler(templateService)
	mentionHandler := handler.NewMentionHandler(mentionService)
	userHandler := handler.NewUserHandler(userService)

	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.Recovery)
	r.Use(middleware.Logging)
	r.Use(middleware.CORS(nil))

	r.Get("/health", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	r.Route("/webhooks", func(r chi.Router) {
		r.Get("/threads", webhookHandler.Verify)
		r.Post("/threads", webhookHandler.Handle)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Get("/threads", authHandler.InitiateOAuth)
			r.Get("/threads/callback", authHandler.Callback)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(authService))
				r.Post("/refresh", authHandler.RefreshToken)
				r.Post("/logout", authHandler.Logout)
				r.Get("/me", authHandler.GetCurrentUser)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))

			r.Route("/user", func(r chi.Router) {
				r.Get("/settings", userHandler.GetSettings)
				r.Patch("/settings", userHandler.UpdateSettings)
				r.Post("/auto-reply/toggle", userHandler.ToggleAutoReply)
				r.Delete("/account", userHandler.DeleteAccount)
			})

			r.Route("/templates", func(r chi.Router) {
				r.Get("/", templateHandler.List)
				r.Post("/", templateHandler.Create)
				r.Get("/{id}", templateHandler.Get)
				r.Put("/{id}", templateHandler.Update)
				r.Delete("/{id}", templateHandler.Delete)
			})

			r.Route("/mentions", func(r chi.Router) {
				r.Get("/", mentionHandler.List)
				r.Post("/sync", mentionHandler.Sync)
				r.Get("/{id}", mentionHandler.Get)
				r.Post("/{id}/retry", mentionHandler.Retry)
			})
		})
	})

	chiLambda = chiadapter.New(r)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return chiLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
