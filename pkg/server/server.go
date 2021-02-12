package server

import (
	"github.com/chef/foodtruck/pkg/storage"
	"github.com/labstack/echo/v4"
)

// Setup initializes an Echo server
func Setup(db storage.Driver, adminAPIKey string, nodesAPIKey string) *echo.Echo {
	e := echo.New()

	initAdminRouter(e, db, adminAPIKey)
	initNodesRouter(e, db, nodesAPIKey)

	return e
}
