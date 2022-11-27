package smtp

type SMTPServer struct {
}

func NewServer() *SMTPServer {
	server := &SMTPServer{}
	return server
}
