package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/gommon/random"
)

type newJobRequestNode struct {
	Org  string `json:"org"`
	Name string `json:"name"`
}
type newJobRequestTask struct {
	WindowStart time.Time              `json:"window_start,omitempty"`
	WindowEnd   time.Time              `json:"window_end,omitempty"`
	Provider    string                 `json:"provider,omitempty"`
	Spec        map[string]interface{} `json:"spec,omitempty"`
}

type newJobRequest struct {
	Nodes []newJobRequestNode `json:"nodes,omitempty"`
	Task  *newJobRequestTask  `json:"task,omitempty"`
}

func randomorg() string {
	return random.String(8, "org"+random.Alphanumeric)
}

func randomnode() string {
	return random.String(8, "node"+random.Alphanumeric)
}

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

func Test_newJob_authorization(t *testing.T) {
	t.Run("unauthorized with random token", func(t *testing.T) {
		asUnauthorized(t).POST("/admin/jobs").
			Expect().
			JSON().
			Path("$.message").
			String().
			Equal("Unauthorized")
	})

	t.Run("unauthorized with nodes token", func(t *testing.T) {
		asNode(t).POST("/admin/jobs").
			Expect().
			Status(http.StatusUnauthorized).
			JSON().
			Path("$.message").
			String().
			Equal("Unauthorized")
	})

	t.Run("authorized with admin token", func(t *testing.T) {
		asAdmin(t).POST("/admin/jobs").
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Object()
	})
}

func Test_newJob_validation(t *testing.T) {
	validNewJobRequest := func() newJobRequest {
		return newJobRequest{
			Nodes: []newJobRequestNode{
				{
					Org:  randomorg(),
					Name: randomnode(),
				},
			},
			Task: &newJobRequestTask{
				WindowStart: time.Now(),
				WindowEnd:   time.Now().AddDate(1, 0, 0),
				Provider:    "some-provider",
			},
		}
	}

	t.Run("Accepts valid request", func(t *testing.T) {
		jobRequest := validNewJobRequest()

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()
	})

	t.Run("Empty object is not valid", func(t *testing.T) {
		asAdmin(t).POST("/admin/jobs").
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String()
	})

	t.Run("Must provide nodes", func(t *testing.T) {
		jobRequest := validNewJobRequest()
		jobRequest.Nodes = []newJobRequestNode{}

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String()
	})

	t.Run("Each node must have an org", func(t *testing.T) {
		jobRequest := validNewJobRequest()
		jobRequest.Nodes = append(jobRequest.Nodes,
			newJobRequestNode{
				Org:  randomorg(),
				Name: randomnode(),
			},
			newJobRequestNode{
				Org:  "",
				Name: randomnode(),
			},
			newJobRequestNode{
				Org:  randomorg(),
				Name: randomnode(),
			},
		)

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String().
			Equal("nodes[2] is not a valid node")
	})

	t.Run("Each node must have an org", func(t *testing.T) {
		jobRequest := validNewJobRequest()
		jobRequest.Nodes = append(jobRequest.Nodes,
			newJobRequestNode{
				Org:  randomorg(),
				Name: randomnode(),
			},
			newJobRequestNode{
				Org:  randomorg(),
				Name: randomnode(),
			},
			newJobRequestNode{
				Org:  randomorg(),
				Name: "",
			},
		)

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String().
			Equal("nodes[3] is not a valid node")
	})

	t.Run("window_start must provided in the task", func(t *testing.T) {
		jobRequest := validNewJobRequest()
		jobRequest.Task.WindowStart = time.Time{}

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String().
			Equal("window_start must be provided")
	})

	t.Run("window_end must provided in the task", func(t *testing.T) {
		jobRequest := validNewJobRequest()
		jobRequest.Task.WindowEnd = time.Time{}

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String().
			Equal("window_end must be provided")
	})

	t.Run("window_end must be after window_start", func(t *testing.T) {
		jobRequest := validNewJobRequest()
		jobRequest.Task.WindowEnd = jobRequest.Task.WindowStart.AddDate(-1, 0, 0)

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String().
			Equal("window_end must be after window_start")
	})

	t.Run("expired tasks are rejected", func(t *testing.T) {
		jobRequest := validNewJobRequest()
		jobRequest.Task.WindowStart = time.Now().AddDate(0, 0, -2)
		jobRequest.Task.WindowEnd = time.Now().AddDate(0, 0, -1)

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String().
			Equal("window has already expired")
	})

	t.Run("provider must be specified", func(t *testing.T) {
		jobRequest := validNewJobRequest()
		jobRequest.Task.Provider = ""

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String().
			Equal("task provider must be provided")
	})

	t.Run("can handle invalid json", func(t *testing.T) {
		asAdmin(t).POST("/admin/jobs").
			WithHeader("Content-Type", "application/json").
			WithBytes([]byte("{\"broken")).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String().
			Equal("invalid request json")
	})
}
