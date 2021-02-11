package test

import (
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/gommon/random"
	"github.com/stretchr/testify/require"
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

type updateNodeTaskStatusResult struct {
	ExitCode int    `json:"exit_code"`
	Reason   string `json:"reason,omitempty"`
}
type updateNodeTaskStatusReq struct {
	JobID  string                      `json:"job_id,omitempty"`
	Status string                      `json:"status,omitempty"`
	Result *updateNodeTaskStatusResult `json:"result,omitempty"`
}

func randomorg() string {
	return random.String(8, "org"+random.Alphanumeric)
}

func randomnode() string {
	return random.String(8, "node"+random.Alphanumeric)
}

func validNewJobRequest(numNodes int) newJobRequest {
	nodes := make([]newJobRequestNode, numNodes)
	for i := range nodes {
		nodes[i] = newJobRequestNode{
			Org:  randomorg(),
			Name: randomnode(),
		}
	}
	return newJobRequest{
		Nodes: nodes,
		Task: &newJobRequestTask{
			WindowStart: time.Now(),
			WindowEnd:   time.Now().AddDate(1, 0, 0),
			Provider:    "some-provider",
		},
	}
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

	t.Run("Accepts valid request", func(t *testing.T) {
		jobRequest := validNewJobRequest(5)

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
		jobRequest := validNewJobRequest(0)

		asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusBadRequest).
			JSON().
			Path("$.message").
			String()
	})

	t.Run("Each node must have an org", func(t *testing.T) {
		jobRequest := validNewJobRequest(1)
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
		jobRequest := validNewJobRequest(1)
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
		jobRequest := validNewJobRequest(1)
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
		jobRequest := validNewJobRequest(1)
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
		jobRequest := validNewJobRequest(1)
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
		jobRequest := validNewJobRequest(1)
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
		jobRequest := validNewJobRequest(1)
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

func Test_getJob(t *testing.T) {
	t.Run("returns not found if the job does not exist", func(t *testing.T) {
		asAdmin(t).GET("/admin/jobs/jobid").
			Expect().
			Status(http.StatusNotFound).
			JSON().
			Path("$.message").
			String().
			Equal("job not found")
	})

	t.Run("returns the job by id without statuses", func(t *testing.T) {
		jobRequest := validNewJobRequest(5)
		jobRequest.Task.Spec = map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": "baz",
			},
		}

		jobID := asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().Path("$.id").String().Raw()

		resp := asAdmin(t).GET("/admin/jobs/{jobID}", jobID).
			Expect().
			Status(http.StatusOK).
			JSON().Object()

		resp.NotContainsKey("statuses")

		expectedTask := *jobRequest.Task
		resp.Path("$.job.id").String().Equal(jobID)
		resp.Path("$.job.task.provider").String().Equal(expectedTask.Provider)
		resp.Path("$.job.task.spec").Object().Equal(expectedTask.Spec)
		requireTimeEquals(t, expectedTask.WindowStart, resp.Path("$.job.task.window_start").String().Raw())
		requireTimeEquals(t, expectedTask.WindowEnd, resp.Path("$.job.task.window_end").String().Raw())
	})

	t.Run("returns the job with statuses", func(t *testing.T) {
		jobRequest := validNewJobRequest(5)
		jobRequest.Task.Spec = map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": "baz",
			},
		}

		jobID := asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().Path("$.id").String().Raw()

		asNode(t).POST(getNextTaskPath(jobRequest.Nodes[0].Org, jobRequest.Nodes[0].Name)).
			Expect().
			Status(http.StatusOK)
		asNode(t).POST(getNextTaskPath(jobRequest.Nodes[1].Org, jobRequest.Nodes[1].Name)).
			Expect().
			Status(http.StatusOK)
		asNode(t).POST(updateTaskStatusPath(jobRequest.Nodes[1].Org, jobRequest.Nodes[1].Name)).
			WithJSON(updateNodeTaskStatusReq{
				JobID:  jobID,
				Status: "running",
			}).
			Expect().
			Status(http.StatusOK)
		asNode(t).POST(getNextTaskPath(jobRequest.Nodes[2].Org, jobRequest.Nodes[2].Name)).
			Expect().
			Status(http.StatusOK)
		asNode(t).POST(updateTaskStatusPath(jobRequest.Nodes[2].Org, jobRequest.Nodes[2].Name)).
			WithJSON(updateNodeTaskStatusReq{
				JobID:  jobID,
				Status: "failed",
				Result: &updateNodeTaskStatusResult{
					ExitCode: 1,
					Reason:   "a reason",
				},
			}).
			Expect().
			Status(http.StatusOK)

		resp := asAdmin(t).GET("/admin/jobs/{jobID}", jobID).
			WithQuery("fetchStatuses", "true").
			Expect().
			Status(http.StatusOK).
			JSON().Object()

		// BUG: https://github.com/chef/foodtruck/issues/11
		// We should have a entry for all the nodes
		resp.Path("$.statuses").Array().Length().Equal(3)

		for _, status := range resp.Path("$.statuses").Array().Iter() {
			// FIXME: https://github.com/chef/foodtruck/issues/12
			// The node_name type is inconsistent with other APIs
			nodeName := status.Object().Path("$.node_name").String().Raw()
			switch nodeName {
			case fmt.Sprintf("%s/%s", jobRequest.Nodes[0].Org, jobRequest.Nodes[0].Name):
				status.Path("$.status").String().Equal("pending")
				requireTimeWithin(t, time.Now(), status.Path("$.last_updated").String().Raw(), 5*time.Second)
				status.Object().NotContainsKey("result")
			case fmt.Sprintf("%s/%s", jobRequest.Nodes[1].Org, jobRequest.Nodes[1].Name):
				status.Path("$.status").String().Equal("running")
				requireTimeWithin(t, time.Now(), status.Path("$.last_updated").String().Raw(), 5*time.Second)
				status.Object().NotContainsKey("result")
			case fmt.Sprintf("%s/%s", jobRequest.Nodes[2].Org, jobRequest.Nodes[2].Name):
				status.Path("$.status").String().Equal("failed")
				requireTimeWithin(t, time.Now(), status.Path("$.last_updated").String().Raw(), 5*time.Second)
				status.Object().ContainsKey("result")
				status.Path("$.result.exit_code").Number().Equal(1)
				status.Path("$.result.reason").String().Equal("a reason")
			default:
				require.Fail(t, "unexpected node name", nodeName)
			}
		}
	})
}

func Test_getNext_authorization(t *testing.T) {
	t.Run("unauthorized with random token", func(t *testing.T) {
		asUnauthorized(t).POST(getNextTaskPath(randomorg(), randomnode())).
			Expect().
			JSON().
			Path("$.message").
			String().
			Equal("Unauthorized")
	})

	t.Run("unauthorized with admin token", func(t *testing.T) {
		asAdmin(t).POST(getNextTaskPath(randomorg(), randomnode())).
			Expect().
			Status(http.StatusUnauthorized).
			JSON().
			Path("$.message").
			String().
			Equal("Unauthorized")
	})

	t.Run("authorized with nodes token", func(t *testing.T) {
		asNode(t).POST(getNextTaskPath(randomorg(), randomnode())).
			Expect().
			Status(http.StatusNotFound).
			JSON().
			Object()
	})
}

func Test_getNext(t *testing.T) {
	t.Run("returns 404 when no tasks are available", func(t *testing.T) {
		asNode(t).POST(getNextTaskPath(randomorg(), randomnode())).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("when there is only one task", func(t *testing.T) {
		jobRequest := validNewJobRequest(1)
		jobRequest.Task.Spec = map[string]interface{}{
			"foo": map[string]string{
				"bar": "baz",
			},
		}

		jobID := asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().Path("$.id").String().Raw()

		resp := asNode(t).POST(getNextTaskPath(jobRequest.Nodes[0].Org, jobRequest.Nodes[0].Name)).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		resp.Path("$.provider").String().Equal(jobRequest.Task.Provider)
		resp.Path("$.spec").Object().Equal(jobRequest.Task.Spec)
		requireTimeEquals(t, jobRequest.Task.WindowStart, resp.Path("$.window_start").String().Raw())
		requireTimeEquals(t, jobRequest.Task.WindowEnd, resp.Path("$.window_end").String().Raw())

		asNode(t).POST(updateTaskStatusPath(jobRequest.Nodes[0].Org, jobRequest.Nodes[0].Name)).
			WithJSON(updateNodeTaskStatusReq{
				JobID:  jobID,
				Status: "success",
			}).
			Expect().
			Status(http.StatusOK)

		asNode(t).POST(getNextTaskPath(jobRequest.Nodes[0].Org, jobRequest.Nodes[0].Name)).
			Expect().
			Status(http.StatusNotFound).
			JSON().
			Object()
	})

	t.Run("returns 404 if the task expires", func(t *testing.T) {
		jobRequest := validNewJobRequest(1)
		jobRequest.Task.WindowEnd = time.Now().Add(100 * time.Millisecond)

		jobID := asAdmin(t).POST("/admin/jobs").
			WithJSON(jobRequest).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().Path("$.id").String().Raw()

		for {
			if time.Now().After(jobRequest.Task.WindowEnd) {
				break
			}
			time.Sleep(jobRequest.Task.WindowEnd.Sub(time.Now()) + 10*time.Millisecond)
		}

		asNode(t).POST(getNextTaskPath(jobRequest.Nodes[0].Org, jobRequest.Nodes[0].Name)).
			Expect().
			Status(http.StatusNotFound).
			JSON().Object()

		resp := asAdmin(t).GET("/admin/jobs/{jobID}", jobID).
			WithQuery("fetchStatuses", "true").
			Expect().
			Status(http.StatusOK).
			JSON().Object()

		resp.Path("$.statuses").Array().Length().Equal(1)
		resp.Path("$.statuses[0].status").String().Equal("expired")
	})

	t.Run("orders tasks by start time", func(t *testing.T) {
		jobRequests := make([]newJobRequest, 10)
		org := randomorg()
		node := randomnode()

		for i := range jobRequests {
			jobRequests[i] = validNewJobRequest(1)
			jobRequests[i].Nodes[0].Org = org
			jobRequests[i].Nodes[0].Name = node
			jobRequests[i].Task.WindowStart = time.Now().Add(time.Duration(-1*len(jobRequests)+i) * time.Hour)
		}

		shuffledJobRequests := make([]newJobRequest, len(jobRequests))
		copy(shuffledJobRequests, jobRequests)
		rand.Shuffle(len(jobRequests), func(i int, j int) {
			shuffledJobRequests[i], shuffledJobRequests[j] = shuffledJobRequests[j], shuffledJobRequests[i]
		})

		jobIDs := make([]string, len(jobRequests))
		for i := range shuffledJobRequests {
			jobID := asAdmin(t).POST("/admin/jobs").
				WithJSON(jobRequests[i]).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object().Path("$.id").String().Raw()

			jobIDs[i] = jobID
		}

		for i := range jobRequests {
			resp := asNode(t).POST(getNextTaskPath(org, node)).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object()

			jobID := resp.Path("$.job_id").String().Raw()
			resp.Path("$.provider").String().Equal(jobRequests[i].Task.Provider)
			requireTimeEquals(t, jobRequests[i].Task.WindowStart, resp.Path("$.window_start").String().Raw())
			requireTimeEquals(t, jobRequests[i].Task.WindowEnd, resp.Path("$.window_end").String().Raw())

			asNode(t).POST(updateTaskStatusPath(org, node)).
				WithJSON(updateNodeTaskStatusReq{
					JobID:  jobID,
					Status: "success",
				}).
				Expect().
				Status(http.StatusOK)
		}
	})
}

func getNextTaskPath(org string, name string) string {
	return fmt.Sprintf("/organizations/%s/foodtruck/nodes/%s/tasks/next", org, name)
}

func updateTaskStatusPath(org string, name string) string {
	return fmt.Sprintf("/organizations/%s/foodtruck/nodes/%s/tasks/status", org, name)
}
