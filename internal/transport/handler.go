package transport

import (
	"downloader/internal/usecase"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type Handler struct{
	service *usecase.DownloadService
}

func NewHandler(service *usecase.DownloadService) *Handler{
	return &Handler{
		service: service,
	}
}

func (h *Handler)StartDownload(w http.ResponseWriter, r *http.Request){
	var req struct {
		URLs    []string `json:"urls"`
		Timeout string   `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil{
		http.Error(w , "invalid format", http.StatusBadRequest)
		return
	}

	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil {
		http.Error(w, "invalid timeout format", http.StatusBadRequest)
		return
	}

	taskID, err := h.service.StartDownload(req.URLs, timeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"task_id": taskID})
}

func (h *Handler) GetDownload(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	taskID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "неверный ID", http.StatusBadRequest)
		return
	}

	task, err := h.service.GetDownload(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *Handler)GetFile (w http.ResponseWriter, r *http.Request){
	taskID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil{
		http.Error(w, "invalid taskID", http.StatusBadRequest)
		return
	}
	fileID, err := strconv.Atoi(r.PathValue("file_id"))
	if err != nil{
		http.Error(w, "invalid fileID", http.StatusBadRequest)
		return
	}
	content, err := h.service.GetFileContent(taskID, fileID)
	if err!= nil{
		http.Error(w, "some problems with content", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(content)
}

func (h *Handler)InitRoutes()(*http.ServeMux){
	mux := http.NewServeMux()
	mux.HandleFunc("POST /downloads", h.StartDownload)
	mux.HandleFunc("GET /downloads/{id}", h.GetDownload)
	mux.HandleFunc("GET /downloads/{id}/files/{file_id}", h.GetFile)
	return mux
}