package main

import (
	"errors"
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
	adminRoutes.GET("/jobs/:job_id", handler.GetJob)
}

type AdminRoutesHandler struct {
	db storage.Driver
}

type AddJobResult struct {
	JobID string `json:"id"`
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

	if job.Task.Provider == "" {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "task provider must be provided"}
	}

	for i, n := range job.Nodes {
		if n.Name == "" || n.Organization == "" {
			return &echo.HTTPError{Code: http.StatusBadRequest, Message: fmt.Sprintf("nodes[%d] is not a valid node", i)}
		}
	}

	jobID, err := h.db.AddJob(c.Request().Context(), job)
	if err != nil {
		return &echo.HTTPError{Code: http.StatusInternalServerError, Internal: err}
	}

	return c.JSON(200, AddJobResult{JobID: jobID})
}

func (h *AdminRoutesHandler) GetJob(c echo.Context) error {
	jobID := c.Param("job_id")

	if jobID == "" {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "must provide a job id"}
	}

	fetchStatuses := c.QueryParam("fetchStatuses") == "true"

	job, err := h.db.GetJob(c.Request().Context(), jobID, storage.WithJobStatuses(fetchStatuses))
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return &echo.HTTPError{Code: http.StatusNotFound, Message: "job not found"}
		}
		return &echo.HTTPError{Code: http.StatusInternalServerError, Internal: err}
	}

	return c.JSON(200, job)
}
