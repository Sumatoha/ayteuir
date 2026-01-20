package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ayteuir/backend/internal/domain"
	"github.com/ayteuir/backend/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

type UpdateSettingsRequest struct {
	ReplyDelaySeconds      int      `json:"reply_delay_seconds"`
	MaxRepliesPerHour      int      `json:"max_replies_per_hour"`
	IgnoreVerifiedAccounts bool     `json:"ignore_verified_accounts"`
	IgnoreKeywords         []string `json:"ignore_keywords"`
}

type ToggleAutoReplyRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *UserHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		if domain.IsNotFound(err) {
			Error(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		Error(w, http.StatusInternalServerError, "FETCH_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"auto_reply_enabled": user.AutoReplyEnabled,
		"settings":           user.Settings,
	})
}

func (h *UserHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.ReplyDelaySeconds < 0 {
		req.ReplyDelaySeconds = 0
	}
	if req.MaxRepliesPerHour <= 0 {
		req.MaxRepliesPerHour = 50
	}
	if req.IgnoreKeywords == nil {
		req.IgnoreKeywords = []string{}
	}

	settings := domain.UserSettings{
		ReplyDelaySeconds:      req.ReplyDelaySeconds,
		MaxRepliesPerHour:      req.MaxRepliesPerHour,
		IgnoreVerifiedAccounts: req.IgnoreVerifiedAccounts,
		IgnoreKeywords:         req.IgnoreKeywords,
	}

	user, err := h.userService.UpdateSettings(r.Context(), userID, settings)
	if err != nil {
		Error(w, http.StatusInternalServerError, "UPDATE_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"auto_reply_enabled": user.AutoReplyEnabled,
		"settings":           user.Settings,
	})
}

func (h *UserHandler) ToggleAutoReply(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	var req ToggleAutoReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	user, err := h.userService.ToggleAutoReply(r.Context(), userID, req.Enabled)
	if err != nil {
		Error(w, http.StatusInternalServerError, "UPDATE_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"auto_reply_enabled": user.AutoReplyEnabled,
	})
}

func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	if err := h.userService.Delete(r.Context(), userID); err != nil {
		Error(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Account deleted successfully"})
}
