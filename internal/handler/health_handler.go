package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/ayteuir/backend/internal/repository/mongodb"
)

type HealthHandler struct {
	mongoClient *mongodb.Client
}

func NewHealthHandler(mongoClient *mongodb.Client) *HealthHandler {
	return &HealthHandler{
		mongoClient: mongoClient,
	}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.mongoClient.Ping(ctx); err != nil {
		Error(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "Database connection failed")
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": "connected",
	})
}
