package whatsapp

import (
	"context"
	"fmt"

	"fasting-bot/internal/config"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type Notifier struct {
	client *whatsmeow.Client
}

func NewNotifier(client *whatsmeow.Client) *Notifier {
	return &Notifier{client: client}
}

func (n *Notifier) Send(jidStr, message string) error {
	if n.client == nil || !n.client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	_, err = n.client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: &message,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (n *Notifier) SendToGroup(message string) error {
	if n.client == nil || !n.client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	if config.AllowedGroupJID == "" {
		return fmt.Errorf("no group JID configured")
	}

	jid, err := types.ParseJID(config.AllowedGroupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %w", err)
	}

	_, err = n.client.SendMessage(context.Background(), jid, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(message),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send group message: %w", err)
	}

	return nil
}
