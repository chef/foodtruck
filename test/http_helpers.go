package test

import (
	"fmt"
	"testing"

	"github.com/gavv/httpexpect/v2"
)

func asAdmin(t *testing.T) *httpexpect.Expect {
	t.Helper()
	return defaultHTTPExpect(t).Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", fmt.Sprintf("Bearer %s", adminAPIKey))
	})
}

func asNode(t *testing.T) *httpexpect.Expect {
	t.Helper()
	return defaultHTTPExpect(t).Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", fmt.Sprintf("Bearer %s", nodesAPIKey))
	})
}

func asUnauthorized(t *testing.T) *httpexpect.Expect {
	t.Helper()
	return defaultHTTPExpect(t).Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", fmt.Sprintf("Bearer %s", "fake-token"))
	})
}

func defaultHTTPExpect(t *testing.T) *httpexpect.Expect {
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  foodtruckServerAddress,
		Reporter: httpexpect.NewRequireReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewCurlPrinter(t),
			httpexpect.NewDebugPrinter(t, true),
		},
	})
}
