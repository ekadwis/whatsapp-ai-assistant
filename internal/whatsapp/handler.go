package whatsapp

import (
	"context"
	"log"
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

	// 1) Abaikan pesan grup dan pesan dari bot sendiri
	if evt.Info.IsGroup || evt.Info.IsFromMe {
		return
	}

	senderUser := evt.Info.Sender.User
	if evt.Info.Chat.User != "" {
		senderUser = evt.Info.Chat.User
	}

	// 2) Ambil isi teks
	text := strings.TrimSpace(getTextFromMessage(evt.Message))
	if text == "" {
		return
	}

	log.Printf("📩 Pesan masuk dari [%s] (Owner: %s): %s", senderUser, h.ownerNumber, text)

	// 3) Whitelist check
	if h.ownerNumber != "" && senderUser != h.ownerNumber {
		log.Printf("⚠️ Pesan diabaikan: Pengirim %s tidak sesuai dengan owner %s", senderUser, h.ownerNumber)
		return
	}

	// 4) Proses pesan ke AI
	log.Println("🤖 Memproses pesan ke AI...")
	response := strings.TrimSpace(h.onMessage(ctx, senderUser, text))
	if response == "" {
		log.Println("⚠️ Respon kosong dari AI")
		return
	}

	// 5) Tentukan target tujuan (gunakan format Chat JID lengkap agar support LID)
	target := evt.Info.Chat.String()
	if target == "" {
		target = evt.Info.Sender.String()
	}

	// Kirim pesan balasan
	err := SendTextWithPresence(ctx, h.messenger, target, response)
	if err != nil {
		log.Printf("❌ Gagal mengirim pesan balasan: %v", err)
		return
	}
	log.Println("✅ Balasan berhasil dikirim!")
}

func getTextFromMessage(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}

	// 1) Regular conversation text
	if t := msg.GetConversation(); t != "" {
		return t
	}

	// 2) Extended text
	if t := msg.GetExtendedTextMessage(); t != nil {
		if text := t.GetText(); text != "" {
			return text
		}
	}

	// 3) Image caption
	if t := msg.GetImageMessage(); t != nil {
		if caption := t.GetCaption(); caption != "" {
			return caption
		}
	}

	// 4) Video caption
	if t := msg.GetVideoMessage(); t != nil {
		if caption := t.GetCaption(); caption != "" {
			return caption
		}
	}

	// 5) Ephemeral/disappearing messages
	if t := msg.GetEphemeralMessage(); t != nil {
		return getTextFromMessage(t.GetMessage())
	}

	// 6) View-once messages
	if t := msg.GetViewOnceMessage(); t != nil {
		return getTextFromMessage(t.GetMessage())
	}

	return ""
}