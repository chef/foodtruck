package main

import (
	"errors"
	"net/http"

	"github.com/chef/foodtruck/pkg/models"
	"github.com/chef/foodtruck/pkg/storage"
	"github.com/labstack/echo/v4"
)

func initNodesRouter(e *echo.Echo, db storage.Driver) {
	handler := &NodeRoutesHandler{
		db: db,
	}
	nodesRoutes := e.Group("/organizations/:org/foodtruck/nodes/:name")
	nodesRoutes.GET("/next", handler.GetNextTask)
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
