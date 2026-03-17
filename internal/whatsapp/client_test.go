package whatsapp

import "testing"

var _ Messenger = (*WhatsAppClient)(nil)

func TestNewWhatsAppClient(t *testing.T) {
	client := NewWhatsAppClient(nil)
	if client == nil {
		t.Fatal("expected NewWhatsAppClient to return non-nil client")
	}
	if client.client != nil {
		t.Fatal("expected underlying whatsmeow client to be nil when initialized with nil")
	}
}
