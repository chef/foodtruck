package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/chef/foodtruck/pkg/foodtruckhttp"
)

var ErrInvalidAuthProvider = errors.New("invalid auth provider type")
var ErrMissingParameters = errors.New("auth provider missing parameter")

type ApiKeyAuthProvider struct {
	Type string `json:"type"`
	Key  string `json:"key"`
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

func (p *ApiKeyAuthProvider) UnmarshalJSON(b []byte) error {
	params := struct {
		Key string `json:"key"`
	}{}
	if err := json.Unmarshal(b, &params); err != nil {
		return err
	}

	p.Key = params.Key

	if p.Key == "" {
		apiKey := os.Getenv("NODES_API_KEY")
		if apiKey != "" {
			p.Key = apiKey
		} else {
			return fmt.Errorf("%w: must provide \"key\"", ErrMissingParameters)
		}
	}
	return nil
}

type AuthConfig struct {
	AuthProvider foodtruckhttp.AuthProvider
}

func (ac *AuthConfig) UnmarshalJSON(b []byte) error {
	providerType := struct {
		Type string `json:"type"`
	}{}

	err := json.Unmarshal(b, &providerType)
	if err != nil {
		return err
	}

	switch providerType.Type {
	case "apiKey":
		p := ApiKeyAuthProvider{}
		if err := json.Unmarshal(b, &p); err != nil {
			return err
		}
		ac.AuthProvider = &p
	default:
		return fmt.Errorf("%w: %q", ErrInvalidAuthProvider, providerType.Type)
	}
	return nil
}
