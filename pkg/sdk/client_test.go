package sdk

import "testing"

func TestNeuroClientAccessors(t *testing.T) {
	client := New(nil, nil, nil, nil, nil)
	if client.Network() != nil || client.Storage() != nil || client.Agent() != nil || client.Identity() != nil || client.Registry() != nil {
		t.Fatal("unexpected nil accessors")
	}
}
