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

type SMTPClient struct {
	Host    string
	Port    uint32
	Timeout time.Duration
	conn    *textproto.Conn
}

func NewClient() *SMTPClient {
	client := &SMTPClient{Port: 25, Timeout: 30 * time.Second}
	return client
}

func (c *SMTPClient) TryConnection(hosts []string) (conn net.Conn, err error) {
	for _, host := range hosts {
		remote := fmt.Sprintf("%s:%d", host, c.Port)
		conn, err := net.DialTimeout("tcp", remote, c.Timeout)
		if err != nil {
			log.Println("try connection", host, c.Port, err)
			continue
		}
		log.Printf("connect %s success\n", remote)
		return conn, nil
	}
	return
}

func (c *SMTPClient) CreateConnection(hostname string) (conn net.Conn, err error) {
	var hosts []string
	if c.Host != "" {
		hosts = []string{c.Host}
	} else {
		hosts = resolveMX(hostname)
		hosts = append(hosts, hostname)
	}
	return c.TryConnection(hosts)
}

func (c *SMTPClient) SetConnection(conn net.Conn) {
	c.conn = textproto.NewConn(conn)
}

func (c *SMTPClient) Quit() {
	c.conn.Cmd("QUIT")
}

func (c *SMTPClient) ExecuteCommand(cmd string, args ...any) func(int) (string, error) {
	id, err := c.conn.Cmd(cmd, args...)
	return func(expectCode int) (string, error) {
		if err != nil {
			return "", err
		}
		c.conn.StartResponse(id)
		defer c.conn.EndResponse(id)
		_, output, err := c.conn.ReadCodeLine(expectCode)
		if err != nil {
			log.Fatal(cmd, "->", err, output)
		}
		return output, err
	}
}

func (c *SMTPClient) Hello() {

}

func (c *SMTPClient) PostMessage(hostname string, from string, recipients []string, content string) error {
	conn, err := c.CreateConnection(hostname)
	if err != nil {
		return err
	}

	c.SetConnection(conn)

	_, err = c.ExecuteCommand("EHLO %s", "localhost")(220)
	if err != nil {
		return err
	}

	_, err = c.ExecuteCommand("MAIL FROM: %s", from)(250)
	if err != nil {
		return err
	}

	for _, rcpt := range recipients {
		_, err = c.ExecuteCommand("RCPT TO:<%s>", rcpt)(250)
		if err != nil {
			return err
		}
	}
	_, err = c.ExecuteCommand("DATA")(250)
	if err != nil {
		return err
	}

	c.conn.W.Write([]byte(content))
	c.conn.W.Write([]byte("\r\n"))
	_, err = c.ExecuteCommand(".")(354)
	if err != nil {
		return err
	}

	c.Close()

	return nil
}

func (c *SMTPClient) Send(msg *Message) {
	for hostname, repts := range groupByHost(msg.GetRecipients()) {
		err := c.PostMessage(hostname, msg.From, repts, msg.ToMime())
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (c *SMTPClient) SendMessage() {
	message := NewMessage()
	c.Send(message)
}

func (c *SMTPClient) Close() error {
	return c.conn.Close()
}
