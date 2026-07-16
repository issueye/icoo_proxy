package main

import (
	"os"
	"testing"

	"github.com/issueye/icoo_proxy/bridge/internal/service"
)

func TestVersion(t *testing.T) {
	want := os.Getenv("EXPECT_BRIDGE_VERSION")
	if want == "" {
		want = "0.0.0-dev"
	}
	if service.Version != want {
		t.Fatalf("service.Version = %q, want %q", service.Version, want)
	}
}
