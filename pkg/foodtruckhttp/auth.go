package foodtruckhttp

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/go-chef/chef"
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

type ChefServerAuthProvider struct {
	conf *chef.AuthConfig
}

func NewChefServerAuthProvider(ClientName string, KeyPath string) (*ChefServerAuthProvider, error) {
	keyData, err := ioutil.ReadFile(KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key at %s: %w", KeyPath, err)
	}
	pk, err := chef.PrivateKeyFromString(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	return &ChefServerAuthProvider{
		conf: &chef.AuthConfig{
			PrivateKey:            pk,
			ClientName:            ClientName,
			AuthenticationVersion: "1.3",
		},
	}, nil
}

func (p *ChefServerAuthProvider) Name() string { return "chefServer" }

func (p *ChefServerAuthProvider) NewPostRequest(requestURL string, body io.Reader) (*http.Request, error) {
	var bodyBytes []byte
	var err error
	if body == nil {
		bodyBytes = []byte("")
	} else {
		bodyBytes, err = ioutil.ReadAll(body)
		if err != nil {
			return nil, err
		}
	}

	shasum := hash256(bodyBytes)

	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ops-Content-Hash", shasum)

	err = p.conf.SignRequest(req)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func hash256(data []byte) string {
	if len(data) == 0 {
		data = []byte("")
	}
	shasum := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(shasum[:])
}
