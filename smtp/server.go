package smtp

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type Handler interface {
	HandleConnection(conn net.Conn)
	HandleCommand(command string, args []string) error
	HandleHELO(domain string) error
	HandleMAIL(from string) error
	HandleRCPT(to string) error
	HandleDATA() error
	HandleQUIT() error
	OnMessage(from string, to []string, data string) error
}

func ListenAndServe(address string, newHandler func(net.Conn) Handler) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Printf("SMTP server listening on %s", address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go newHandler(conn).HandleConnection(conn)
	}
}

type DefaultHandler struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer

	From string
	To   []string
	Data string
}

func NewDefaultHandler(conn net.Conn) *DefaultHandler {
	return &DefaultHandler{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}
}

func (h *DefaultHandler) SendResponse(code int, message string) error {
	_, err := fmt.Fprintf(h.writer, "%d %s\r\n", code, message)
	if err != nil {
		return err
	}
	return h.writer.Flush()
}

func (h *DefaultHandler) HandleConnection(conn net.Conn) {
	defer conn.Close()
	h.conn = conn
	h.reader = bufio.NewReader(conn)
	h.writer = bufio.NewWriter(conn)
	h.SendResponse(220, "SMTP Server Ready")
	for {
		line, err := h.reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading from connection: %v", err)
			return
		}
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) == 0 {
			continue
		}
		command := strings.ToUpper(parts[0])
		args := parts[1:]
		if err := h.HandleCommand(command, args); err != nil {
			log.Printf("Error handling command: %v", err)
			h.SendResponse(550, "Error handling command")
		}
		if command == "QUIT" {
			return
		}
	}
}

func (h *DefaultHandler) HandleCommand(command string, args []string) error {
	switch command {
	case "HELO", "EHLO":
		if len(args) < 1 {
			return h.SendResponse(501, "Syntax error in parameters or arguments")
		}
		return h.HandleHELO(args[0])
	case "MAIL":
		if len(args) < 1 || !strings.HasPrefix(strings.ToUpper(args[0]), "FROM:") {
			return h.SendResponse(501, "Syntax error in parameters or arguments")
		}
		return h.HandleMAIL(strings.TrimPrefix(args[0], "FROM:"))
	case "RCPT":
		if len(args) < 1 || !strings.HasPrefix(strings.ToUpper(args[0]), "TO:") {
			return h.SendResponse(501, "Syntax error in parameters or arguments")
		}
		return h.HandleRCPT(strings.TrimPrefix(args[0], "TO:"))
	case "DATA":
		return h.HandleDATA()
	case "QUIT":
		return h.HandleQUIT()
	default:
		return h.SendResponse(500, "Command not recognized")
	}
}

func (h *DefaultHandler) HandleHELO(domain string) error {
	return h.SendResponse(250, fmt.Sprintf("Hello %s", domain))
}

func (h *DefaultHandler) HandleMAIL(from string) error {
	h.From = from
	return h.SendResponse(250, "OK")
}

func (h *DefaultHandler) HandleRCPT(to string) error {
	h.To = append(h.To, to)
	return h.SendResponse(250, "OK")
}

func (h *DefaultHandler) HandleDATA() (err error) {
	var messageData strings.Builder
	if err := h.SendResponse(354, "Start mail input; end with <CRLF>.<CRLF>"); err != nil {
		return err
	}
	for {
		line, err := h.reader.ReadString('\n')
		if err != nil {
			return err
		}
		if line == ".\r\n" {
			break
		}
		messageData.WriteString(line)
	}
	h.Data = messageData.String()
	h.OnMessage(h.From, h.To, h.Data)
	return
}

func (h *DefaultHandler) HandleQUIT() error {
	return h.SendResponse(221, "Bye")
}

func (h *DefaultHandler) OnMessage(from string, to []string, data string) error {
	log.Printf("Received message: From=%s, To=%v, Data=%s", from, to, data)
	return h.SendResponse(250, "OK")
}
