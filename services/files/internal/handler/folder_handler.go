package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ivanastamenkovic/file-storage-sharing-platform/services/files/internal/model"
	"github.com/ivanastamenkovic/file-storage-sharing-platform/services/files/internal/service"
)

type FolderHandler struct {
	svc *service.FolderService
}

func NewFolderHandler(svc *service.FolderService) *FolderHandler {
	return &FolderHandler{svc: svc}
}

func (h *FolderHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/files/folders", h.Create)
}

type createFolderRequest struct {
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}

func (h *FolderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID := userID(r)
	if ownerID == "" {
		writeError(w, model.ErrUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req createFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.ErrInvalidRequest)
		return
	}

	folder, err := h.svc.CreateFolder(r.Context(), ownerID, req.ParentID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, folder)
}

func userID(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		log.Printf("internal error: %v", err)
		appErr = &model.AppError{Status: 500, Code: "INTERNAL_ERROR", Message: "Internal server error"}
	}
	writeJSON(w, appErr.Status, map[string]string{
		"error":   appErr.Code,
		"message": appErr.Message,
	})
}
