package subsystems

import (
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/MortalArena/Musketeers/pkg/acp"
)

type MessagingSubsystem struct {
	acpRouter    *acp.Router
	acpTransport *acp.Transport
	topics       map[string]*pubsub.Topic
	mu           sync.RWMutex
	chunkAsm     any
}

func NewMessagingSubsystem(acpRouter *acp.Router, acpTransport *acp.Transport) *MessagingSubsystem {
	return &MessagingSubsystem{acpRouter: acpRouter, acpTransport: acpTransport, topics: make(map[string]*pubsub.Topic)}
}

func (s *MessagingSubsystem) SetACP(router *acp.Router, transport *acp.Transport) {
	s.acpRouter = router
	s.acpTransport = transport
}

func (s *MessagingSubsystem) ACPRouter() *acp.Router           { return s.acpRouter }
func (s *MessagingSubsystem) ACPTransport() *acp.Transport     { return s.acpTransport }
func (s *MessagingSubsystem) ChunkAssembler() any              { return s.chunkAsm }
func (s *MessagingSubsystem) Lock()                            { s.mu.Lock() }
func (s *MessagingSubsystem) Unlock()                          { s.mu.Unlock() }
func (s *MessagingSubsystem) Topics() map[string]*pubsub.Topic { return s.topics }
func (s *MessagingSubsystem) SetTopic(name string, topic *pubsub.Topic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.topics[name] = topic
}
func (s *MessagingSubsystem) Topic(name string) (*pubsub.Topic, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	topic, exists := s.topics[name]
	return topic, exists
}
