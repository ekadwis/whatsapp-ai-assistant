package whatsapp

import (
	"context"
	"testing"

	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type mockMessenger struct {
	sendTextCalls int
	lastRecipient string
	lastText      string
}

func (m *mockMessenger) SendText(ctx context.Context, recipient string, text string) error {
	m.sendTextCalls++
	m.lastRecipient = recipient
	m.lastText = text
	return nil
}

func (m *mockMessenger) SendPresence(ctx context.Context, recipient string) error {
	return nil
}

func TestGetTextFromMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  *waE2E.Message
		want string
	}{
		{
			name: "plain conversation",
			msg: &waE2E.Message{
				Conversation: proto.String("halo"),
			},
			want: "halo",
		},
		{
			name: "extended text",
			msg: &waE2E.Message{
				ExtendedTextMessage: &waE2E.ExtendedTextMessage{
					Text: proto.String("teks panjang"),
				},
			},
			want: "teks panjang",
		},
		{
			name: "image caption",
			msg: &waE2E.Message{
				ImageMessage: &waE2E.ImageMessage{
					Caption: proto.String("caption gambar"),
				},
			},
			want: "caption gambar",
		},
		{
			name: "video caption",
			msg: &waE2E.Message{
				VideoMessage: &waE2E.VideoMessage{
					Caption: proto.String("caption video"),
				},
			},
			want: "caption video",
		},
		{
			name: "ephemeral wraps conversation",
			msg: &waE2E.Message{
				EphemeralMessage: &waE2E.FutureProofMessage{
					Message: &waE2E.Message{
						Conversation: proto.String("pesan ephemeral"),
					},
				},
			},
			want: "pesan ephemeral",
		},
		{
			name: "view once wraps conversation",
			msg: &waE2E.Message{
				ViewOnceMessage: &waE2E.FutureProofMessage{
					Message: &waE2E.Message{
						Conversation: proto.String("pesan view once"),
					},
				},
			},
			want: "pesan view once",
		},
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
		{
			name: "empty message",
			msg:  &waE2E.Message{},
			want: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := getTextFromMessage(tc.msg)
			if got != tc.want {
				t.Fatalf("getTextFromMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHandler_WhitelistAndMessageFlow(t *testing.T) {
	t.Parallel()

	makeEvent := func(sender string, isGroup bool, msg *waE2E.Message) *events.Message {
		return &events.Message{
			Info: types.MessageInfo{
				MessageSource: types.MessageSource{
					Sender: types.JID{
						User:   sender,
						Server: types.DefaultUserServer,
					},
					IsGroup: isGroup,
				},
			},
			Message: msg,
		}
	}

	t.Run("non-owner is silently ignored", func(t *testing.T) {
		m := &mockMessenger{}
		callbackCalls := 0

		h := NewHandler(m, "6281234567890", func(ctx context.Context, sender, text string) string {
			callbackCalls++
			return "ok"
		})

		h.handleMessage(context.Background(), makeEvent(
			"6289999999999",
			false,
			&waE2E.Message{Conversation: proto.String("beli ayam 16k")},
		))

		if callbackCalls != 0 {
			t.Fatalf("callback should not be called for non-owner, got %d", callbackCalls)
		}
		if m.sendTextCalls != 0 {
			t.Fatalf("SendText should not be called for non-owner, got %d", m.sendTextCalls)
		}
	})

	t.Run("group message is ignored", func(t *testing.T) {
		m := &mockMessenger{}
		callbackCalls := 0

		h := NewHandler(m, "6281234567890", func(ctx context.Context, sender, text string) string {
			callbackCalls++
			return "ok"
		})

		h.handleMessage(context.Background(), makeEvent(
			"6281234567890",
			true,
			&waE2E.Message{Conversation: proto.String("halo grup")},
		))

		if callbackCalls != 0 {
			t.Fatalf("callback should not be called for group messages, got %d", callbackCalls)
		}
		if m.sendTextCalls != 0 {
			t.Fatalf("SendText should not be called for group messages, got %d", m.sendTextCalls)
		}
	})

	t.Run("empty text does not trigger callback", func(t *testing.T) {
		m := &mockMessenger{}
		callbackCalls := 0

		h := NewHandler(m, "6281234567890", func(ctx context.Context, sender, text string) string {
			callbackCalls++
			return "ok"
		})

		h.handleMessage(context.Background(), makeEvent(
			"6281234567890",
			false,
			&waE2E.Message{},
		))

		if callbackCalls != 0 {
			t.Fatalf("callback should not be called for empty text, got %d", callbackCalls)
		}
		if m.sendTextCalls != 0 {
			t.Fatalf("SendText should not be called for empty text, got %d", m.sendTextCalls)
		}
	})

	t.Run("owner private message triggers callback and response", func(t *testing.T) {
		m := &mockMessenger{}
		callbackCalls := 0

		h := NewHandler(m, "6281234567890", func(ctx context.Context, sender, text string) string {
			callbackCalls++
			if sender != "6281234567890" {
				t.Fatalf("unexpected sender: %q", sender)
			}
			if text != "beli ayam 16k" {
				t.Fatalf("unexpected text: %q", text)
			}
			return "✅ dicatat"
		})

		h.handleMessage(context.Background(), makeEvent(
			"6281234567890",
			false,
			&waE2E.Message{Conversation: proto.String("beli ayam 16k")},
		))

		if callbackCalls != 1 {
			t.Fatalf("callback should be called once, got %d", callbackCalls)
		}
		if m.sendTextCalls != 1 {
			t.Fatalf("SendText should be called once, got %d", m.sendTextCalls)
		}
		if m.lastRecipient != "6281234567890" {
			t.Fatalf("unexpected recipient: %q", m.lastRecipient)
		}
		if m.lastText != "✅ dicatat" {
			t.Fatalf("unexpected text sent: %q", m.lastText)
		}
	})

	t.Run("empty callback response does not send message", func(t *testing.T) {
		m := &mockMessenger{}
		callbackCalls := 0

		h := NewHandler(m, "6281234567890", func(ctx context.Context, sender, text string) string {
			callbackCalls++
			return ""
		})

		h.handleMessage(context.Background(), makeEvent(
			"6281234567890",
			false,
			&waE2E.Message{Conversation: proto.String("tes")},
		))

		if callbackCalls != 1 {
			t.Fatalf("callback should be called once, got %d", callbackCalls)
		}
		if m.sendTextCalls != 0 {
			t.Fatalf("SendText should not be called when response is empty, got %d", m.sendTextCalls)
		}
	})
}
