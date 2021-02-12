package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
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

func requireTimeEquals(t *testing.T, goTime time.Time, stringTime string) {
	requireTimeWithin(t, goTime, stringTime, time.Second)
}

func requireTimeWithin(t *testing.T, goTime time.Time, stringTime string, dur time.Duration) {
	convertedTime, err := time.Parse(time.RFC3339, stringTime)
	require.NoError(t, err, "failed to parse time")
	require.WithinDuration(t, goTime, convertedTime, time.Second)
}
