package foodtruckhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/chef/foodtruck/pkg/models"
)

type Client struct {
	BaseURL      string
	Node         models.Node
	httpClient   *http.Client
	authProvider AuthProvider
}

type AuthProvider interface {
	Name() string
	NewPostRequest(requestURL string, body io.Reader) (*http.Request, error)
}

func NewClient(baseURL string, node models.Node, authProvider AuthProvider) *Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
	}
	return &Client{
		BaseURL:      fmt.Sprintf("%s/organizations/%s/foodtruck/nodes/%s", baseURL, node.Organization, node.Name),
		Node:         node,
		authProvider: authProvider,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   time.Duration(5*time.Second) * time.Second,
		},
	}
}

func (c *Client) GetNextTask(ctx context.Context) (models.NodeTask, error) {
	resp, err := c.post(ctx, "/tasks/next", nil)
	defer resp.Body.Close()
	if err != nil {
		return models.NodeTask{}, err
	}
	if resp.StatusCode == 200 {
		d := json.NewDecoder(resp.Body)
		task := models.NodeTask{}
		if err := d.Decode(&task); err != nil {
			return models.NodeTask{}, err
		}
		return task, nil
	} else if resp.StatusCode == 404 {
		return models.NodeTask{}, models.ErrNoTasks
	}
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Fprintf(os.Stderr, "Unknown response:\n%s\n\n", body)
	return models.NodeTask{}, fmt.Errorf("Request failed")
}

func (c *Client) UpdateNodeTaskStatus(ctx context.Context, nodeTaskStatus models.NodeTaskStatus) error {
	reqBody, err := json.Marshal(nodeTaskStatus)
	resp, err := c.post(ctx, "/tasks/status", bytes.NewReader(reqBody))
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		return nil
	}
	respBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Fprintf(os.Stderr, "Unknown response:\n%s\n\n", respBody)
	return fmt.Errorf("Request failed")
}

func (c *Client) post(ctx context.Context, requestURL string, body io.Reader) (*http.Response, error) {
	u := c.BaseURL + requestURL
	req, err := c.authProvider.NewPostRequest(u, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
