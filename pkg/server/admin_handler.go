package server

import (
	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
}

func (h *AdminHandler) ListJobs(c echo.Context) error {
	return nil
}
