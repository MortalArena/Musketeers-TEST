package messaging

import (
	"context"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

func TestMessagingCapabilitySendAndJoin(t *testing.T) {
	messenger := NewMemoryMessenger()
	capability := NewMessagingCapability(messenger, messenger)
	if _, err := capability.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, JoinChannelCommand{Channel: "general"}); err != nil {
		t.Fatalf("Join Execute returned error: %v", err)
	}
	if _, err := capability.Execute(context.Background(), policy.Principal{DID: "did:ia:test"}, SendMessageCommand{Channel: "general", Message: "hello"}); err != nil {
		t.Fatalf("Send Execute returned error: %v", err)
	}
	if len(messenger.Channels()) != 1 || messenger.Channels()[0] != "general" {
		t.Fatalf("unexpected channels: %v", messenger.Channels())
	}
	if len(messenger.Messages()) != 1 || messenger.Messages()[0].Message != "hello" {
		t.Fatalf("unexpected messages: %#v", messenger.Messages())
	}
}
