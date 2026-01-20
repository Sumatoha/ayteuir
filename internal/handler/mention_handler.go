package handler

import (
	"net/http"
	"strconv"

	"github.com/ayteuir/backend/internal/domain"
	"github.com/ayteuir/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MentionHandler struct {
	mentionService *service.MentionService
}

func NewMentionHandler(mentionService *service.MentionService) *MentionHandler {
	return &MentionHandler{
		mentionService: mentionService,
	}
}

func (h *MentionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	mentions, err := h.mentionService.GetMentions(r.Context(), userID, limit, offset)
	if err != nil {
		Error(w, http.StatusInternalServerError, "FETCH_ERROR", err.Error())
		return
	}

	Paginated(w, mentions, limit, offset)
}

func (h *MentionHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	mentionID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_MENTION_ID", "Invalid mention ID")
		return
	}

	mention, err := h.mentionService.GetMention(r.Context(), userID, mentionID)
	if err != nil {
		if domain.IsNotFound(err) {
			Error(w, http.StatusNotFound, "NOT_FOUND", "Mention not found")
			return
		}
		if domain.IsForbidden(err) {
			Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
			return
		}
		Error(w, http.StatusInternalServerError, "FETCH_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, mention)
}

func (h *MentionHandler) Retry(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	mentionID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_MENTION_ID", "Invalid mention ID")
		return
	}

	if err := h.mentionService.RetryMention(r.Context(), userID, mentionID); err != nil {
		if domain.IsNotFound(err) {
			Error(w, http.StatusNotFound, "NOT_FOUND", "Mention not found")
			return
		}
		if domain.IsForbidden(err) {
			Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
			return
		}
		Error(w, http.StatusBadRequest, "RETRY_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Retry initiated"})
}

// Sync manually pulls mentions from Threads API (fallback when webhooks not working)
func (h *MentionHandler) Sync(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	result, err := h.mentionService.PullMentions(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SYNC_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}
