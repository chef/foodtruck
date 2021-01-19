package main

import (
	"fmt"
	"net/http"

	"github.com/chef/foodtruck/pkg/models"
	"github.com/chef/foodtruck/pkg/storage"
	"github.com/labstack/echo/v4"
)

func initAdminRouter(e *echo.Echo, db storage.Driver) {
	handler := &AdminRoutesHandler{
		db: db,
	}
	adminRoutes := e.Group("/admin")
	adminRoutes.POST("/jobs", handler.AddJob)
}

type AdminRoutesHandler struct {
	db storage.Driver
}

func (h *AdminRoutesHandler) AddJob(c echo.Context) error {
	job := models.Job{}
	if err := c.Bind(&job); err != nil {
		return err
	}

	if len(job.Nodes) == 0 {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "no nodes provided"}
	}

	if job.Task.WindowStart.IsZero() || job.Task.WindowEnd.IsZero() {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "window_start and window_end must be provided"}
	}

	if job.Task.Type == "" {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "task type must be provided"}
	}

	for i, n := range job.Nodes {
		if n.Name == "" || n.Organization == "" {
			return &echo.HTTPError{Code: http.StatusBadRequest, Message: fmt.Sprintf("nodes[%d] is not a valid node", i)}
		}
	}

	if err := h.db.AddJob(c.Request().Context(), job); err != nil {
		return &echo.HTTPError{Code: http.StatusInternalServerError, Internal: err}
	}
	return nil
}
