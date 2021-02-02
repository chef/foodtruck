package foodtruckhttp

import (
	"io"
	"net/http"
)

type ApiKeyAuthProvider struct {
	Key string
}

func (p *ApiKeyAuthProvider) Name() string { return "apiKey" }

func (p *ApiKeyAuthProvider) NewPostRequest(requestURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest("POST", requestURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.Key)

	return req, nil
}
