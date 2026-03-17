package whatsapp

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

type Handler struct {
	messenger   Messenger
	ownerNumber string
	onMessage   func(ctx context.Context, sender string, text string) string
}

func NewHandler(
	m Messenger,
	ownerNumber string,
	onMessage func(ctx context.Context, sender string, text string) string,
) *Handler {
	if onMessage == nil {
		onMessage = func(context.Context, string, string) string { return "" }
	}

	return &Handler{
		messenger:   m,
		ownerNumber: ownerNumber,
		onMessage:   onMessage,
	}
}

func (h *Handler) Register(client *whatsmeow.Client) {
	if client == nil {
		return
	}

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			go h.handleMessage(context.Background(), v)
		}
	})
}

func (h *Handler) handleMessage(ctx context.Context, evt *events.Message) {
	if evt == nil {
		return
	}

	// 1) Whitelist check FIRST (silent reject).
	if evt.Info.Sender.User != h.ownerNumber {
		return
	}

	// 2) Ignore groups in V1.
	if evt.Info.IsGroup {
		return
	}

	// 3) Extract text from all supported variants.
	text := strings.TrimSpace(getTextFromMessage(evt.Message))
	if text == "" {
		return
	}

	// 4) Run callback.
	response := strings.TrimSpace(h.onMessage(ctx, evt.Info.Sender.User, text))
	if response == "" {
		return
	}

	// 5) Send response with typing presence (best-effort, silent on error).
	_ = SendTextWithPresence(ctx, h.messenger, evt.Info.Sender.User, response)
}

func getTextFromMessage(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}

	// 1) Regular conversation text.
	if t := msg.GetConversation(); t != "" {
		return t
	}

	// 2) Extended text (links/web client/quoted text containers).
	if t := msg.GetExtendedTextMessage(); t != nil {
		if text := t.GetText(); text != "" {
			return text
		}
	}

	// 3) Image caption.
	if t := msg.GetImageMessage(); t != nil {
		if caption := t.GetCaption(); caption != "" {
			return caption
		}
	}

	// 4) Video caption.
	if t := msg.GetVideoMessage(); t != nil {
		if caption := t.GetCaption(); caption != "" {
			return caption
		}
	}

	// 5) Ephemeral/disappearing messages.
	if t := msg.GetEphemeralMessage(); t != nil {
		return getTextFromMessage(t.GetMessage())
	}

	// 6) View-once messages.
	if t := msg.GetViewOnceMessage(); t != nil {
		return getTextFromMessage(t.GetMessage())
	}

	return ""
}
