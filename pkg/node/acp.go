package node

import (
	"context"
	"encoding/json"

	"github.com/libp2p/go-libp2p/core/peer"
	libp2pproto "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/MortalArena/Musketeers/pkg/acp"
)

// initACP يهيئ بروتوكول ACP على العقدة
func (n *Node) initACP() {
	router := acp.NewRouter()
	transport := acp.NewTransport(n.host(), n.keyPair().DID, n.keyPair().Private, n, router, n.log)
	n.messaging.SetACP(router, transport)
	n.host().SetStreamHandler(libp2pproto.ID(acp.ProtocolID), transport.ServeStream)
}

// ACPRouter يرجع موجّه مهام ACP
func (n *Node) ACPRouter() *acp.Router {
	return n.messaging.ACPRouter()
}

// RegisterACPTask يسجّل معالج مهمة مخصص
func (n *Node) RegisterACPTask(task string, handler acp.TaskHandler) {
	n.messaging.ACPRouter().Register(task, handler)
}

// SendACPTask يرسل مهمة ACP لنظير
func (n *Node) SendACPTask(ctx context.Context, pid peer.ID, toDID, task string, input interface{}) (*acp.Envelope, error) {
	var raw json.RawMessage
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	return n.messaging.ACPTransport().SendTask(ctx, pid, toDID, task, raw, "")
}

// SupportedACPTasks يرجع المهام المدعومة محلياً
func (n *Node) SupportedACPTasks() []string {
	return n.messaging.ACPRouter().SupportedTasks()
}
