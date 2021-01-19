package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func initAdminRouter(e *echo.Echo) {
	adminRoutes := e.Group("/admin")
	adminRoutes.GET("/jobs/list", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
}
