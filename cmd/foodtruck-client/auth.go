package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/chef/foodtruck/pkg/foodtruckhttp"
)

var ErrInvalidAuthProvider = errors.New("invalid auth provider type")
var ErrMissingParameters = errors.New("auth provider missing parameter")

type AuthProviderFactory interface {
	InitializeAuthProvider(nodeName string) (foodtruckhttp.AuthProvider, error)
}

type apiKeyAuthProviderFactory struct {
	Key string `json:"key"`
}

func (p *apiKeyAuthProviderFactory) InitializeAuthProvider(nodeName string) (foodtruckhttp.AuthProvider, error) {
	return &foodtruckhttp.ApiKeyAuthProvider{
		Key: p.Key,
	}, nil
}

func (p *apiKeyAuthProviderFactory) UnmarshalJSON(b []byte) error {
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
	AuthProvider AuthProviderFactory
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
		p := apiKeyAuthProviderFactory{}
		if err := json.Unmarshal(b, &p); err != nil {
			return err
		}
		ac.AuthProvider = &p
	default:
		return fmt.Errorf("%w: %q", ErrInvalidAuthProvider, providerType.Type)
	}
	return nil
}
