package smtp

import (
	"fmt"
	"log"
	"net"
	"net/textproto"
	"sort"
	"strings"
	"time"
)

func parseAddress(address string) (string, string) {
	s := strings.Split(address, "@")
	return s[0], s[1]
}

func groupByHost(recipients []string) map[string][]string {
	output := make(map[string][]string)
	for _, recipient := range recipients {
		_, hostname := parseAddress(recipient)
		output[hostname] = append(output[hostname], recipient)
	}
	return output
}

func resolveMX(hostname string) []string {
	records, err := net.LookupMX(hostname)
	checkError(err)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Pref < records[j].Pref
	})
	hosts := []string{}
	for _, record := range records {
		hosts = append(hosts, record.Host)
	}
	return hosts
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// SMTPClient represents an SMTP client
type SMTPClient struct {
	Host    string
	Port    uint32
	Timeout time.Duration
	conn    *textproto.Conn
}

// NewClient creates a new SMTPClient with specified settings
func NewClient() *SMTPClient {
	return &SMTPClient{
		Port:    25,
		Timeout: 30 * time.Second,
	}
}

// CreateConnection establishes a connection to an SMTP server
func (c *SMTPClient) CreateConnection(hostname string) (net.Conn, error) {
	var hosts []string
	if c.Host != "" {
		hosts = []string{c.Host}
	} else {
		hosts = resolveMX(hostname)
		hosts = append(hosts, hostname)
	}
	return c.tryConnection(hosts)
}

// tryConnection attempts to connect to each host in the list
func (c *SMTPClient) tryConnection(hosts []string) (net.Conn, error) {
	for _, host := range hosts {
		remote := fmt.Sprintf("%s:%d", host, c.Port)
		conn, err := net.DialTimeout("tcp", remote, c.Timeout)
		if err != nil {
			continue
		}
		return conn, nil
	}
	return nil, fmt.Errorf("failed to connect to any host")
}

// SetConnection sets the textproto connection
func (c *SMTPClient) SetConnection(conn net.Conn) {
	c.conn = textproto.NewConn(conn)
}

// ExecuteCommand sends a command and returns a function to read the response
func (c *SMTPClient) ExecuteCommand(cmd string, args ...any) func(int) (string, error) {
	id, err := c.conn.Cmd(cmd, args...)
	return func(expectCode int) (string, error) {
		if err != nil {
			return "", err
		}
		c.conn.StartResponse(id)
		defer c.conn.EndResponse(id)
		_, output, err := c.conn.ReadCodeLine(expectCode)
		return output, err
	}
}

// PostMessage sends an email message
func (c *SMTPClient) PostMessage(hostname, from string, recipients []string, content string) error {
	conn, err := c.CreateConnection(hostname)
	if err != nil {
		return err
	}
	c.SetConnection(conn)
	defer c.Close()

	if _, err := c.ExecuteCommand("EHLO %s", "localhost")(220); err != nil {
		return err
	}

	if _, err := c.ExecuteCommand("MAIL FROM: %s", from)(250); err != nil {
		return err
	}

	for _, rcpt := range recipients {
		if _, err := c.ExecuteCommand("RCPT TO:<%s>", rcpt)(250); err != nil {
			return err
		}
	}

	if _, err := c.ExecuteCommand("DATA")(354); err != nil {
		return err
	}

	if _, err := c.conn.W.Write([]byte(content + "\r\n")); err != nil {
		return err
	}

	if _, err := c.ExecuteCommand(".")(250); err != nil {
		return err
	}

	return nil
}

// Send sends a Message
func (c *SMTPClient) Send(msg *Message) error {
	for hostname, repts := range groupByHost(msg.GetRecipients()) {
		if err := c.PostMessage(hostname, msg.From, repts, msg.ToMime()); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the connection
func (c *SMTPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
