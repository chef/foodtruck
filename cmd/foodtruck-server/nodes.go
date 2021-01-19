package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/chef/foodtruck/pkg/models"
	"github.com/chef/foodtruck/pkg/storage"
	"github.com/labstack/echo/v4"
)

func initNodesRouter(e *echo.Echo, db storage.Driver) {
	handler := &NodeRoutesHandler{
		db: db,
	}
	nodesRoutes := e.Group("/organizations/:org/foodtruck/nodes/:name")
	nodesRoutes.PUT("/tasks/next", handler.GetNextTask)
	nodesRoutes.PUT("/tasks/status", handler.UpdateNodeTaskStatus)
}

type NodeRoutesHandler struct {
	db storage.Driver
}

func (h *NodeRoutesHandler) GetNextTask(c echo.Context) error {
	node, err := nodeFromContext(c)
	if err != nil {
		return err
	}
	task, err := h.db.NextNodeTask(c.Request().Context(), node)
	if err != nil {
		if errors.Is(err, models.ErrNoTasks) {
			return &echo.HTTPError{Code: http.StatusNotFound, Message: "no tasks available"}
		}
		return &echo.HTTPError{Code: http.StatusInternalServerError, Internal: err}
	}

	return c.JSON(http.StatusOK, task)
}

func (h *NodeRoutesHandler) UpdateNodeTaskStatus(c echo.Context) error {
	node, err := nodeFromContext(c)
	if err != nil {
		return err
	}
	body := models.NodeTaskStatus{}
	if err := c.Bind(&body); err != nil {
		return err
	}

	if body.JobID == "" {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: fmt.Sprintf("job_id must be provided")}
	}
	if !models.IsValidTaskStatus(string(body.Status)) {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: fmt.Sprintf("status must be one of (%s)", strings.Join(models.ValidTaskStatuses, ","))}
	}
	err = h.db.UpdateNodeTaskStatus(c.Request().Context(), node, body)
	if err != nil {
		if errors.Is(err, models.ErrNoTasks) {
			return &echo.HTTPError{Code: http.StatusNotFound, Message: "no tasks available"}
		}
		return &echo.HTTPError{Code: http.StatusInternalServerError, Internal: err}
	}

	return nil
}

func nodeFromContext(c echo.Context) (models.Node, error) {
	org := c.Param("org")
	name := c.Param("name")
	if org == "" || name == "" {
		return models.Node{}, &echo.HTTPError{Code: http.StatusBadRequest, Message: "invalid node"}
	}
	return models.Node{
		Organization: org,
		Name:         name,
	}, nil
}
