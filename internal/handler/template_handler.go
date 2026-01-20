package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ayteuir/backend/internal/domain"
	"github.com/ayteuir/backend/internal/middleware"
	"github.com/ayteuir/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateHandler struct {
	templateService *service.TemplateService
}

func NewTemplateHandler(templateService *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
	}
}

type CreateTemplateRequest struct {
	Name        string             `json:"name"`
	MentionType domain.MentionType `json:"mention_type"`
	Content     string             `json:"content"`
}

type UpdateTemplateRequest struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	IsActive bool   `json:"is_active"`
	Priority int    `json:"priority"`
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	templates, err := h.templateService.GetAll(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "FETCH_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, templates)
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Name == "" || req.Content == "" || req.MentionType == "" {
		Error(w, http.StatusBadRequest, "MISSING_FIELDS", "Name, content, and mention_type are required")
		return
	}

	template, err := h.templateService.Create(r.Context(), userID, req.Name, req.MentionType, req.Content)
	if err != nil {
		Error(w, http.StatusInternalServerError, "CREATE_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusCreated, template)
}

func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	templateID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_TEMPLATE_ID", "Invalid template ID")
		return
	}

	template, err := h.templateService.GetByID(r.Context(), userID, templateID)
	if err != nil {
		if domain.IsNotFound(err) {
			Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found")
			return
		}
		if domain.IsForbidden(err) {
			Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
			return
		}
		Error(w, http.StatusInternalServerError, "FETCH_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, template)
}

func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	templateID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_TEMPLATE_ID", "Invalid template ID")
		return
	}

	var req UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	template, err := h.templateService.Update(r.Context(), userID, templateID, req.Name, req.Content, req.IsActive, req.Priority)
	if err != nil {
		if domain.IsNotFound(err) {
			Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found")
			return
		}
		if domain.IsForbidden(err) {
			Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
			return
		}
		Error(w, http.StatusInternalServerError, "UPDATE_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, template)
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	templateID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_TEMPLATE_ID", "Invalid template ID")
		return
	}

	if err := h.templateService.Delete(r.Context(), userID, templateID); err != nil {
		if domain.IsNotFound(err) {
			Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found")
			return
		}
		if domain.IsForbidden(err) {
			Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied")
			return
		}
		Error(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Template deleted successfully"})
}

func getUserID(r *http.Request) (primitive.ObjectID, error) {
	userIDStr := middleware.GetUserID(r.Context())
	return primitive.ObjectIDFromHex(userIDStr)
}
