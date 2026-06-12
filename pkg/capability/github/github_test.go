package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

func TestGitHubCapabilityListRepos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/octo/repos" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"repo1"}]`))
	}))
	defer server.Close()
	capability := &GitHubCapability{BaseURL: server.URL, HTTPClient: server.Client()}
	result, err := capability.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, ListReposCommand{Owner: "octo"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	repos := result.Output["repos"].([]map[string]any)
	if repos[0]["name"] != "repo1" {
		t.Fatalf("unexpected repos: %#v", repos)
	}
}

func TestGitHubCapabilityCreateIssueValidation(t *testing.T) {
	capability := &GitHubCapability{}
	_, err := capability.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, CreateIssueCommand{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
