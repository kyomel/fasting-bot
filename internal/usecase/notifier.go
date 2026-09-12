package usecase

// Notifier is the outbound messaging port. Infrastructure adapters (WhatsApp,
// future SMS/telegram/email) implement this so the scheduler stays independent
// of any specific delivery mechanism.
type Notifier interface {
	Send(jid, message string) error
	SendToGroup(message string) error
}
