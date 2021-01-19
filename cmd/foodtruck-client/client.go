package main

import (
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
	"github.com/davecgh/go-spew/spew"
)

type Config struct {
	Node          models.Node   `json:"node"`
	BaseURL       string        `json:"base_url"`
	ProvidersPath string        `json:"providers_path"`
	Interval      time.Duration `json:"interval"`
}

type Client struct {
	BaseURL    string
	Node       models.Node
	httpClient *http.Client
}

func NewClient(baseURL string, node models.Node) *Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
	}
	return &Client{
		BaseURL: fmt.Sprintf("%s/organizations/%s/foodtruck/nodes/%s", baseURL, node.Organization, node.Name),
		Node:    node,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   time.Duration(5*time.Second) * time.Second,
		},
	}
}

func (c *Client) GetNextTask(ctx context.Context) (models.NodeTask, error) {
	resp, err := c.put(ctx, "/tasks/next", nil)
	if err != nil {
		return models.NodeTask{}, err
	}
	defer resp.Body.Close()
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

func (c *Client) put(ctx context.Context, requestURL string, body io.ReadCloser) (*http.Response, error) {
	u := c.BaseURL + requestURL
	req, err := http.NewRequest("PUT", u, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func main() {
	config := Config{
		BaseURL: "http://localhost:1323",
		Node: models.Node{
			Organization: "neworg",
			Name:         "testnode5",
		},
		Interval: time.Second * 5,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := NewClient(config.BaseURL, config.Node)
	for {
		select {
		case <-ctx.Done():
			break
		case <-time.After(config.Interval):
			task, err := client.GetNextTask(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[Error]: %s\n", err)
				continue
			}
			spew.Dump(task)
		}
	}
}
