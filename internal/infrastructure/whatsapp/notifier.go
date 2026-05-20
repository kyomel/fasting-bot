package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
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
		return fmt.Errorf("invalid JID %s: %w", jidStr, err)
	}

	_, err = n.client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: &message,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
