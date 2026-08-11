package transport

import (
	"downloader/internal/domain"
	"downloader/internal/usecase"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	service *usecase.DownloadService
}

func NewHandler(service *usecase.DownloadService) *Handler {
	return &Handler{
		service: service,
	}
}

func handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrClient) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, domain.ErrBusiness) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func (h *Handler) StartDownload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Files []struct {
			URL string `json:"url"`
		} `json:"files"`
		Timeout string `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, domain.ErrClient)
		return
	}

	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil {
		handleError(w, domain.ErrClient)
		return
	}

	var urls []string
	for _, f := range req.Files {
		urls = append(urls, f.URL)
	}

	taskID, err := h.service.StartDownload(r.Context(), urls, timeout)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":     taskID,
		"status": "PROCESS",
	})
}

func (h *Handler) GetDownload(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	taskID, err := strconv.Atoi(idStr)
	if err != nil {
		handleError(w, domain.ErrClient)
		return
	}

	task, err := h.service.GetDownload(r.Context(), taskID)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		handleError(w, domain.ErrClient)
		return
	}
	fileID, err := strconv.Atoi(r.PathValue("file_id"))
	if err != nil {
		handleError(w, domain.ErrClient)
		return
	}
	content, err := h.service.GetFileContent(r.Context(), taskID, fileID)
	if err != nil {
		handleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(content)
}

func (h *Handler) InitRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /downloads", h.StartDownload)
	mux.HandleFunc("GET /downloads/{id}", h.GetDownload)
	mux.HandleFunc("GET /downloads/{id}/files/{file_id}", h.GetFile)
	return mux
}
