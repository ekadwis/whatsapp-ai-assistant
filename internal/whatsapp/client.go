package whatsapp

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Messenger abstracts WhatsApp message sending for testability.
type Messenger interface {
	SendText(ctx context.Context, recipient string, text string) error
	SendPresence(ctx context.Context, recipient string) error
}

type WhatsAppClient struct {
	client *whatsmeow.Client
	log    waLog.Logger
}

func NewWhatsAppClient(client *whatsmeow.Client) *WhatsAppClient {
	return &WhatsAppClient{
		client: client,
	}
}

func (w *WhatsAppClient) SendText(ctx context.Context, recipient string, text string) error {
	jid, err := types.ParseJID(recipient + "@s.whatsapp.net")
	if err != nil {
		return fmt.Errorf("invalid JID %s: %w", recipient, err)
	}

	_, err = w.client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		return err
	}

	return nil
}

func (w *WhatsAppClient) SendPresence(ctx context.Context, recipient string) error {
	jid, err := types.ParseJID(recipient + "@s.whatsapp.net")
	if err != nil {
		return fmt.Errorf("invalid JID %s: %w", recipient, err)
	}

	w.client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)

	delay := time.Duration(500+rand.Intn(1001)) * time.Millisecond
	time.Sleep(delay)

	w.client.SendChatPresence(ctx, jid, types.ChatPresencePaused, types.ChatPresenceMediaText)

	return nil
}
