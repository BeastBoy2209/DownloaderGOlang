package transport

import (
	"downloader/internal/domain"
	"downloader/internal/usecase"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

type Handler struct {
	service *usecase.DownloadService
}

func NewHandler(service *usecase.DownloadService) *Handler {
	return &Handler{
		service: service,
	}
}

func handleError(c *echo.Context, err error) error {
	if errors.Is(err, domain.ErrClient) {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	if errors.Is(err, domain.ErrBusiness) {
		return c.JSON(422, map[string]string{"error": err.Error()})
	}
	log.Printf("request failed: %v", err)
	return c.JSON(500, map[string]string{"error": "internal server error"})
}

func (h *Handler) StartDownload(c *echo.Context) error {
	var req struct {
		Files []struct {
			URL string `json:"url"`
		} `json:"files"`
		Timeout string `json:"timeout"`
	}
	if err := c.Bind(&req); err != nil {
		return handleError(c, domain.ErrClient)
	}

	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil {
		return handleError(c, domain.ErrClient)
	}

	var urls []string
	for _, f := range req.Files {
		urls = append(urls, f.URL)
	}

	taskID, err := h.service.StartDownload(c.Request().Context(), urls, timeout)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, map[string]any{
		"id":     taskID,
		"status": "PROCESS",
	})
}

func (h *Handler) GetDownload(c *echo.Context) error {
	idStr := c.Param("id")
	taskID, err := strconv.Atoi(idStr)
	if err != nil {
		return handleError(c, domain.ErrClient)
	}

	task, err := h.service.GetDownload(c.Request().Context(), taskID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(200, task)
}

func (h *Handler) GetFile(c *echo.Context) error {
	taskID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handleError(c, domain.ErrClient)
	}
	fileID, err := strconv.Atoi(c.Param("file_id"))
	if err != nil {
		return handleError(c, domain.ErrClient)
	}
	content, err := h.service.GetFileContent(c.Request().Context(), taskID, fileID)
	if err != nil {
		return handleError(c, err)
	}
	return c.Blob(200, "application/octet-stream", content)
}

func (h *Handler) InitRoutes() *echo.Echo {
	e := echo.New()
	e.POST("/downloads", h.StartDownload)
	e.GET("/downloads/:id", h.GetDownload)
	e.GET("/downloads/:id/files/:file_id", h.GetFile)
	return e
}
