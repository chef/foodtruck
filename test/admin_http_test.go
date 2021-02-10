package test

import (
	"net/http"
	"testing"
)

func Test_getJob_authorization(t *testing.T) {
	t.Run("unauthorized with random token", func(t *testing.T) {
		asUnauthorized(t).GET("/admin/jobs/jobid").
			Expect().
			JSON().
			Path("$.message").
			String().
			Equal("Unauthorized")
	})

	t.Run("unauthorized with nodes token", func(t *testing.T) {
		asNode(t).GET("/admin/jobs/jobid").
			Expect().
			Status(http.StatusUnauthorized).
			JSON().
			Path("$.message").
			String().
			Equal("Unauthorized")
	})

	t.Run("authorized with admin token", func(t *testing.T) {
		asAdmin(t).GET("/admin/jobs/jobid").
			Expect().
			Status(http.StatusNotFound).
			JSON().
			Object()
	})
}
