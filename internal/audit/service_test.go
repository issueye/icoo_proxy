package audit

import (
	"os"
	"path/filepath"
	"testing"

	"icoo_proxy/internal/appdb"
)

func TestServiceAddAndList(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer os.Chdir(oldwd)

	svc := &Service{}
	defer svc.Close()

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := svc.Add(RequestLogInput{
		Method:          "POST",
		Path:            "/v1/chat/completions",
		Model:           "gpt-4o-mini",
		TargetModel:     "gpt-4o-mini",
		ProviderID:      "openai-main",
		ProviderName:    "OpenAI",
		ProviderType:    "openai",
		EndpointMode:    "responses",
		UpstreamBase:    "https://api.openai.com/v1",
		UpstreamPath:    "/responses",
		Streaming:       true,
		StatusCode:      200,
		DurationMs:      123,
		ClientIP:        "127.0.0.1:3000",
		UserAgent:       "unit-test",
		RequestPayload:  `{"model":"gpt-4o-mini"}`,
		ResponseHeaders: `{"Content-Type":["application/json"]}`,
		ResponsePayload: `{"id":"chatcmpl-test","choices":[]}`,
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	records, err := svc.List(10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d", len(records))
	}
	if records[0].Model != "gpt-4o-mini" {
		t.Fatalf("Model = %q", records[0].Model)
	}
	if records[0].ProviderID != "openai-main" {
		t.Fatalf("ProviderID = %q", records[0].ProviderID)
	}
	if records[0].EndpointMode != "responses" {
		t.Fatalf("EndpointMode = %q", records[0].EndpointMode)
	}
	if records[0].UpstreamPath != "/responses" {
		t.Fatalf("UpstreamPath = %q", records[0].UpstreamPath)
	}
	if records[0].StatusCode != 200 {
		t.Fatalf("StatusCode = %d", records[0].StatusCode)
	}
	if records[0].RequestPayload == "" {
		t.Fatalf("expected request payload to be persisted")
	}
	if records[0].ResponseHeaders == "" {
		t.Fatalf("expected response headers to be persisted")
	}
	if records[0].ResponsePayload == "" {
		t.Fatalf("expected response payload to be persisted")
	}

	if _, err := os.Stat(filepath.Join(dir, filepath.Base(appdb.DBPath()))); err != nil {
		t.Fatalf("expected db to exist: %v", err)
	}
}
