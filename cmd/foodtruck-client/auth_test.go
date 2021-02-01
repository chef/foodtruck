package main

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthConfigInvalid(t *testing.T) {
	ac := AuthConfig{}
	err := unmarshal(`{"type": "invalid"}`, &ac)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalidAuthProvider))
}

func TestAuthConfigApiKey(t *testing.T) {
	t.Run("missing key parameter", func(t *testing.T) {
		ac := AuthConfig{}
		err := unmarshal(`{"type": "apiKey"}`, &ac)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrMissingParameters))
	})

	t.Run("key in json", func(t *testing.T) {
		ac := AuthConfig{}
		err := unmarshal(`{"type": "apiKey", "key": "asdf"}`, &ac)
		require.NoError(t, err)
		require.Equal(t, "apiKey", ac.AuthProvider.Name())
	})

	t.Run("key in environment", func(t *testing.T) {
		os.Setenv("NODES_API_KEY", "asdf") // nolint: errcheck
		defer os.Unsetenv("NODES_API_KEY")

		ac := AuthConfig{}
		err := unmarshal(`{"type": "apiKey"}`, &ac)
		require.NoError(t, err)
		require.Equal(t, "apiKey", ac.AuthProvider.Name())
	})

	t.Run("key in json", func(t *testing.T) {
		ac := AuthConfig{}
		err := unmarshal(`{"type": "apiKey", "key": "asdf"}`, &ac)
		require.NoError(t, err)
		require.Equal(t, "apiKey", ac.AuthProvider.Name())
	})

}

func unmarshal(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
