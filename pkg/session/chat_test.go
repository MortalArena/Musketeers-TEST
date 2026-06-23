package session

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/dgraph-io/badger/v4"
)

func TestChatManager(t *testing.T) {
	eventBus := eventbus.NewEventBus()
	cm := NewChatManager("test-session", eventBus)

	t.Run("AddMessage", func(t *testing.T) {
		msg := ChatMessage{
			Type:    MsgTypeMessage,
			Content: "Test message",
			Source:  "human",
		}
		cm.AddMessage(msg)

		messages := cm.GetMessages()
		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}
	})

	t.Run("AddPermanentMessage", func(t *testing.T) {
		msg := ChatMessage{
			Type:    MsgTypeMessage,
			Content: "Important goal",
			Source:  "human",
		}
		err := cm.AddPermanentMessage(msg)
		if err != nil {
			t.Errorf("Failed to add permanent message: %v", err)
		}

		permanent := cm.GetPermanentMessages()
		if len(permanent) != 1 {
			t.Errorf("Expected 1 permanent message, got %d", len(permanent))
		}
	})

	t.Run("GetMessagesByType", func(t *testing.T) {
		msg1 := ChatMessage{Type: MsgTypeThought, Content: "Thinking", Source: "agent"}
		msg2 := ChatMessage{Type: MsgTypeMessage, Content: "Hello", Source: "human"}
		cm.AddMessage(msg1)
		cm.AddMessage(msg2)

		thoughts := cm.GetMessagesByType(MsgTypeThought)
		if len(thoughts) < 1 {
			t.Errorf("Expected at least 1 thought message")
		}
	})

	t.Run("SessionFolder", func(t *testing.T) {
		folder := cm.GetSessionFolder()
		if folder == "" {
			t.Error("Expected session folder to be set")
		}
	})
}

func TestCollectiveMemory(t *testing.T) {
	// إنشاء DB مؤقت للاختبار
	opts := badger.DefaultOptions("").WithInMemory(true)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("Failed to open badger DB: %v", err)
	}
	defer db.Close()

	mem := NewCollectiveMemory("test-session", db)

	t.Run("AddKnowledge", func(t *testing.T) {
		item := KnowledgeItem{
			Type:        "file",
			Name:        "requirements.md",
			Description: "Project requirements",
			Content:     "# Requirements",
			Category:    "requirements",
			Priority:    10,
		}
		err := mem.AddKnowledge(item)
		if err != nil {
			t.Errorf("Failed to add knowledge: %v", err)
		}

		if mem.TotalKnowledge != 1 {
			t.Errorf("Expected 1 knowledge item, got %d", mem.TotalKnowledge)
		}
	})

	t.Run("GetKnowledgeByCategory", func(t *testing.T) {
		items := mem.GetKnowledgeByCategory("requirements")
		if len(items) != 1 {
			t.Errorf("Expected 1 item in requirements category, got %d", len(items))
		}
	})

	t.Run("GetKnowledgeByPriority", func(t *testing.T) {
		items := mem.GetKnowledgeByPriority(5)
		if len(items) != 1 {
			t.Errorf("Expected 1 item with priority >= 5, got %d", len(items))
		}
	})

	t.Run("SearchKnowledge", func(t *testing.T) {
		items := mem.SearchKnowledge("requirements")
		if len(items) != 1 {
			t.Errorf("Expected 1 item matching 'requirements', got %d", len(items))
		}
	})
}

func TestMessageAttachment(t *testing.T) {
	t.Run("AttachmentCreation", func(t *testing.T) {
		attachment := &MessageAttachment{
			Type:      "image",
			Name:      "screenshot.png",
			Size:      1024,
			MimeType:  "image/png",
			FilePath:  "/sessions/test/attachments/screenshot.png",
			Processed: true,
		}

		if attachment.Type != "image" {
			t.Error("Expected attachment type to be image")
		}
		if !attachment.Processed {
			t.Error("Expected attachment to be processed")
		}
	})
}
