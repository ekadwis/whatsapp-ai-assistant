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
	SendImage(ctx context.Context, recipient string, imageBytes []byte, caption string) error
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

func (w *WhatsAppClient) SendImage(ctx context.Context, recipient string, imageBytes []byte, caption string) error {
	if len(imageBytes) == 0 {
		return fmt.Errorf("image bytes is empty")
	}

	jid, err := types.ParseJID(recipient + "@s.whatsapp.net")
	if err != nil {
		return fmt.Errorf("invalid JID %s: %w", recipient, err)
	}

	uploaded, err := w.client.Upload(ctx, imageBytes, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image to whatsapp: %w", err)
	}

	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String("image/png"),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageBytes))),
		},
	}

	_, err = w.client.SendMessage(ctx, jid, msg)
	return err
}
