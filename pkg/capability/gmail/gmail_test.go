package gmail

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

func TestGmailCapabilityListEmails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"messages":[{"id":"1"}]}`))
	}))
	defer server.Close()
	capability := &GmailCapability{BaseURL: server.URL, HTTPClient: server.Client()}
	result, err := capability.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, ListEmailsCommand{Max: 1})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	messages := result.Output["messages"].([]map[string]any)
	if messages[0]["id"] != "1" {
		t.Fatalf("unexpected messages: %#v", messages)
	}
}

func TestGmailCapabilitySendValidation(t *testing.T) {
	capability := &GmailCapability{}
	_, err := capability.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, SendEmailCommand{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
