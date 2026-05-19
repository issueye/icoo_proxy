package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/service"
)

func TestContainerInitializesAndCloses(t *testing.T) {
	container := newTestContainer(t)

	if container.DB == nil {
		t.Fatal("DB is nil")
	}
	if container.TrafficDB == nil {
		t.Fatal("TrafficDB is nil")
	}
	if container.Config.DBPath == container.Config.TrafficDBPath {
		t.Fatalf("traffic DB should be independent, got shared path %q", container.Config.DBPath)
	}
	if container.DB.Migrator().HasTable("traffic_records") {
		t.Fatal("main DB should not contain traffic_records")
	}
	if !container.TrafficDB.Migrator().HasTable("traffic_records") {
		t.Fatal("traffic DB should contain traffic_records")
	}
	if container.Server == nil || container.Server.Handler == nil {
		t.Fatal("server handler is nil")
	}

	if err := container.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestProviderAdminCRUD(t *testing.T) {
	container := newTestContainer(t)
	t.Cleanup(func() {
		if err := container.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	const adminSecret = "admin-secret"
	if _, err := container.Services.Auth.CreateKey(context.Background(), service.APIKeyCreateInput{
		Name:    "admin",
		Secret:  adminSecret,
		Scopes:  "admin",
		Enabled: true,
	}); err != nil {
		t.Fatalf("create admin key: %v", err)
	}

	createBody := map[string]any{
		"id":          "provider-test",
		"name":        "Test Provider",
		"protocol":    "openai-chat",
		"vendor":      "custom",
		"base_url":    "https://example.test/v1",
		"api_key":     "provider-secret",
		"enabled":     true,
		"description": "created by test",
	}
	createResp := doJSON(t, container.Server.Handler, http.MethodPost, "/api/v1/providers", adminSecret, createBody)
	if createResp.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", createResp.Code, createResp.Body.String())
	}
	assertResponseDataField(t, createResp.Body.Bytes(), "ID", "provider-test")

	listResp := doJSON(t, container.Server.Handler, http.MethodGet, "/api/v1/providers", adminSecret, nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listResp.Code, listResp.Body.String())
	}
	assertResponseDataContainsID(t, listResp.Body.Bytes(), "provider-test")

	updateBody := map[string]any{
		"name":        "Updated Provider",
		"protocol":    "anthropic",
		"vendor":      "anthropic",
		"base_url":    "https://anthropic.example.test",
		"api_key":     "updated-secret",
		"enabled":     true,
		"description": "updated by test",
	}
	updateResp := doJSON(t, container.Server.Handler, http.MethodPut, "/api/v1/providers/provider-test", adminSecret, updateBody)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateResp.Code, updateResp.Body.String())
	}
	assertResponseDataField(t, updateResp.Body.Bytes(), "Name", "Updated Provider")

	deleteResp := doJSON(t, container.Server.Handler, http.MethodDelete, "/api/v1/providers/provider-test", adminSecret, nil)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleteResp.Code, deleteResp.Body.String())
	}
}

func TestCustomEndpointRoutesProxyRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("upstream path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer upstream-secret" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_test","object":"response","model":"gpt-5.4","status":"completed","output":[],"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}`))
	}))
	defer upstream.Close()

	container := newTestContainer(t)
	t.Cleanup(func() {
		if err := container.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if _, err := container.Services.Auth.CreateKey(context.Background(), service.APIKeyCreateInput{
		Name:    "proxy",
		Secret:  "proxy-secret",
		Scopes:  "proxy",
		Enabled: true,
	}); err != nil {
		t.Fatalf("create proxy key: %v", err)
	}
	if _, err := container.Services.Provider.Upsert(context.Background(), service.ProviderUpsertInput{
		ID:       "provider-openai",
		Name:     "OpenAI",
		Protocol: constants.ProtocolOpenAIResponses,
		Vendor:   constants.VendorOpenAI,
		BaseURL:  upstream.URL,
		APIKey:   "upstream-secret",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if _, err := container.Services.ProviderModel.Upsert(context.Background(), service.ProviderModelUpsertInput{
		ProviderID: "provider-openai",
		Name:       "gpt-5.4",
		MaxTokens:  32768,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("create provider model: %v", err)
	}
	if _, err := container.Services.RoutingRule.Upsert(context.Background(), service.RoutingRuleUpsertInput{
		Name:              "custom responses",
		Priority:          1,
		MatchProtocol:     constants.ProtocolOpenAIResponses,
		MatchModelPattern: "*",
		TargetProviderID:  "provider-openai",
		TargetModel:       "gpt-5.4",
		Enabled:           true,
	}); err != nil {
		t.Fatalf("create routing rule: %v", err)
	}
	if _, err := container.Services.Endpoint.Upsert(context.Background(), service.EndpointUpsertInput{
		Path:               "/responses",
		DownstreamProtocol: constants.ProtocolOpenAIResponses,
		Enabled:            true,
	}); err != nil {
		t.Fatalf("create custom endpoint: %v", err)
	}

	resp := doJSON(t, container.Server.Handler, http.MethodPost, "/responses", "proxy-secret", map[string]any{
		"model": "gpt-5.4",
		"input": "hello",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("custom endpoint status = %d, body = %s", resp.Code, resp.Body.String())
	}

	records, err := container.Services.Traffic.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list traffic: %v", err)
	}
	if len(records) != 1 || records[0].Endpoint != "/responses" || records[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected traffic records: %+v", records)
	}
}

func TestAPIKeySecretEndpointAndMetadataUpdate(t *testing.T) {
	container := newTestContainer(t)
	t.Cleanup(func() {
		if err := container.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	const adminSecret = "admin-secret"
	if _, err := container.Services.Auth.CreateKey(context.Background(), service.APIKeyCreateInput{
		Name:    "admin",
		Secret:  adminSecret,
		Scopes:  "admin",
		Enabled: true,
	}); err != nil {
		t.Fatalf("create admin key: %v", err)
	}

	createResp := doJSON(t, container.Server.Handler, http.MethodPost, "/api/v1/api-keys", adminSecret, map[string]any{
		"name":    "copyable",
		"secret":  "copyable-secret",
		"scopes":  "proxy",
		"enabled": true,
	})
	if createResp.Code != http.StatusOK {
		t.Fatalf("create api key status = %d, body = %s", createResp.Code, createResp.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.Data.ID == "" {
		t.Fatal("created key id is empty")
	}

	secretResp := doJSON(t, container.Server.Handler, http.MethodGet, "/api/v1/api-keys/"+created.Data.ID+"/secret", adminSecret, nil)
	if secretResp.Code != http.StatusOK {
		t.Fatalf("get secret status = %d, body = %s", secretResp.Code, secretResp.Body.String())
	}
	assertResponseDataField(t, secretResp.Body.Bytes(), "secret", "copyable-secret")

	updateResp := doJSON(t, container.Server.Handler, http.MethodPost, "/api/v1/api-keys", adminSecret, map[string]any{
		"id":      created.Data.ID,
		"name":    "copyable-updated",
		"secret":  "",
		"scopes":  "admin,proxy",
		"enabled": true,
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update api key status = %d, body = %s", updateResp.Code, updateResp.Body.String())
	}

	secretResp = doJSON(t, container.Server.Handler, http.MethodGet, "/api/v1/api-keys/"+created.Data.ID+"/secret", adminSecret, nil)
	if secretResp.Code != http.StatusOK {
		t.Fatalf("get secret after update status = %d, body = %s", secretResp.Code, secretResp.Body.String())
	}
	assertResponseDataField(t, secretResp.Body.Bytes(), "secret", "copyable-secret")
}

func newTestContainer(t *testing.T) *Container {
	t.Helper()

	dataDir := t.TempDir()
	configPath := filepath.Join(dataDir, "config.toml")
	config := []byte("allow_local_without_auth = false\n")
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	container, err := NewContainer(Options{
		ConfigPath:   configPath,
		DataDir:      dataDir,
		AddrOverride: "127.0.0.1:18182",
	})
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	return container
}

func doJSON(t *testing.T, handler http.Handler, method string, path string, apiKey string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("x-api-key", apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func assertResponseDataField(t *testing.T, raw []byte, field string, want string) {
	t.Helper()

	var response struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got, _ := response.Data[field].(string); got != want {
		t.Fatalf("data.%s = %q, want %q", field, got, want)
	}
}

func assertResponseDataContainsID(t *testing.T, raw []byte, wantID string) {
	t.Helper()

	var response struct {
		Data struct {
			Items    []map[string]any `json:"items"`
			Total    int              `json:"total"`
			Page     int              `json:"page"`
			PageSize int              `json:"page_size"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Data.Total == 0 || response.Data.Page != 1 {
		t.Fatalf("unexpected page metadata: %+v", response.Data)
	}
	for _, item := range response.Data.Items {
		if got, _ := item["ID"].(string); got == wantID {
			return
		}
	}
	t.Fatalf("response data does not contain ID %q", wantID)
}
