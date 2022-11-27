package smtp

import "strings"

type Message struct {
	From    string
	To      string
	Cc      string
	Bcc     string
	Subject string
	Content string
}

func NewMessage() *Message {
	message := &Message{}
	return message
}

func (m *Message) GetRecipients() (recipients []string) {
	if m.To != "" {
		recipients = append(recipients, m.To)
	}
	if m.Cc != "" {
		recipients = append(recipients, m.Cc)
	}
	if m.Bcc != "" {
		recipients = append(recipients, m.Bcc)
	}
	return
}

func (m *Message) ToMime() (content string) {
	builder := strings.Builder{}
	builder.WriteString("From: " + m.From + "\n")
	builder.WriteString("To: " + m.To + "\n")
	builder.WriteString("Subject: " + m.Subject + "\n")
	builder.WriteString("\n")
	builder.WriteString(m.Content)
	return builder.String()
}
