package vault

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKV_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/secret/data/my-app/config", r.URL.Path)

		// Verify request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]interface{}
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)

		// KV v2 wraps data in "data" field
		data, ok := payload["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value1", data["key1"])
		assert.Equal(t, "value2", data["key2"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	data := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	err = kv.Put(context.Background(), "my-app/config", data)
	assert.NoError(t, err)
}

func TestKV_PutWithMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]interface{}
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)

		// Verify metadata
		options, ok := payload["options"].(map[string]interface{})
		assert.True(t, ok)

		customMeta, ok := options["custom_metadata"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value1", customMeta["meta1"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	data := map[string]interface{}{
		"key": "value",
	}

	metadata := map[string]string{
		"meta1": "value1",
	}

	err = kv.PutWithMetadata(context.Background(), "test/path", data, metadata)
	assert.NoError(t, err)
}

func TestKV_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/secret/data/my-app/config", r.URL.Path)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"username": "admin",
					"password": "secret123",
				},
				"metadata": map[string]interface{}{
					"version": 1,
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	secret, err := kv.Get(context.Background(), "my-app/config")
	require.NoError(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, "admin", secret.Data["username"])
	assert.Equal(t, "secret123", secret.Data["password"])
}

func TestKV_Get_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	secret, err := kv.Get(context.Background(), "nonexistent/path")
	assert.Error(t, err)
	assert.Nil(t, secret)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestKV_GetVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		// Check version parameter
		version := r.URL.Query().Get("version")
		assert.Equal(t, "2", version)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"key": "old-value",
				},
				"metadata": map[string]interface{}{
					"version": 2,
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	secret, err := kv.GetVersion(context.Background(), "test/path", 2)
	require.NoError(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, "old-value", secret.Data["key"])
}

func TestKV_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/secret/delete/my-app/config", r.URL.Path)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	err = kv.Delete(context.Background(), "my-app/config")
	assert.NoError(t, err)
}

func TestKV_DeleteVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/secret/delete/my-app/config", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]interface{}
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)

		versions := payload["versions"].([]interface{})
		assert.Len(t, versions, 2)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	err = kv.DeleteVersions(context.Background(), "my-app/config", []int{1, 2})
	assert.NoError(t, err)
}

func TestKV_Destroy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/secret/destroy/my-app/config", r.URL.Path)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	err = kv.Destroy(context.Background(), "my-app/config", []int{1})
	assert.NoError(t, err)
}

func TestKV_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "LIST", r.Method)
		assert.Equal(t, "/v1/secret/metadata/my-app", r.URL.Path)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"keys": []string{"config1", "config2", "config3"},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	keys, err := kv.List(context.Background(), "my-app")
	require.NoError(t, err)
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "config1")
	assert.Contains(t, keys, "config2")
	assert.Contains(t, keys, "config3")
}

func TestKV_List_EmptyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	keys, err := kv.List(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestKV_GetMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/secret/metadata/my-app/config", r.URL.Path)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"version":      3,
				"created_time": "2024-01-01T00:00:00Z",
				"custom_metadata": map[string]string{
					"owner": "admin",
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	metadata, err := kv.GetMetadata(context.Background(), "my-app/config")
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, 3, metadata.Version)
	assert.Equal(t, "2024-01-01T00:00:00Z", metadata.CreatedTime)
}

func TestKV_GetMetadata_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	kv := client.KV()

	metadata, err := kv.GetMetadata(context.Background(), "nonexistent/path")
	assert.Error(t, err)
	assert.Nil(t, metadata)
	assert.Contains(t, err.Error(), "secret not found")
}
