package examples

import (
	"github.com/song940/smtp-go/smtp"
)

func RunClient() {
	client := smtp.NewClient()
	client.Host = "localhost"
	client.Port = 2525
	message := smtp.NewMessage()
	message.From = "song940@gmail.com"
	message.To = "song940@gmail.com"
	message.Subject = "Test Email"
	message.Content = "This is a test message"
	defer client.Close()
	client.Send(message)
}
