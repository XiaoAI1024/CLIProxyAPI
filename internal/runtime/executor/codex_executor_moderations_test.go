package executor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestCodexExecutorModerationsPassthrough(t *testing.T) {
	var gotPath string
	var gotBody []byte
	var gotAuth string
	var gotOriginator string
	var gotAccountID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotOriginator = r.Header.Get("Originator")
		gotAccountID = r.Header.Get("Chatgpt-Account-Id")
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"modr_1","model":"omni-moderation-latest","results":[{"flagged":false}]}`))
	}))
	defer server.Close()

	executor := NewCodexExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{
		Provider: "codex",
		Attributes: map[string]string{
			"base_url": server.URL + "/v1",
		},
		Metadata: map[string]any{
			"access_token": "test-access-token",
			"email":        "user@example.com",
			"account_id":   "acct_123",
		},
	}

	resp, err := executor.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "omni-moderation-latest",
		Payload: []byte(`{"input":"hello","model":"omni-moderation-latest"}`),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FromString("openai"),
		Alt:          "moderations",
		Stream:       false,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if gotPath != "/v1/moderations" {
		t.Fatalf("path = %q, want %q", gotPath, "/v1/moderations")
	}
	if gotAuth != "Bearer test-access-token" {
		t.Fatalf("authorization = %q, want %q", gotAuth, "Bearer test-access-token")
	}
	if gotOriginator == "" {
		t.Fatal("expected Originator header to be set")
	}
	if gotAccountID != "acct_123" {
		t.Fatalf("chatgpt account id = %q, want %q", gotAccountID, "acct_123")
	}
	if got := gjson.GetBytes(gotBody, "model").String(); got != "omni-moderation-latest" {
		t.Fatalf("model = %q, want %q", got, "omni-moderation-latest")
	}
	if got := gjson.GetBytes(gotBody, "input").String(); got != "hello" {
		t.Fatalf("input = %q, want %q", got, "hello")
	}
	if string(resp.Payload) != `{"id":"modr_1","model":"omni-moderation-latest","results":[{"flagged":false}]}` {
		t.Fatalf("payload = %s", string(resp.Payload))
	}
}
