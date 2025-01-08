package main

import (
	"fmt"
	"log"
	"net"

	"github.com/lsongdev/smtp-go/smtp"
)

type MyHandler struct {
	*smtp.DefaultHandler
}

func (h *MyHandler) OnMessage(from string, to []string, data string) error {
	fmt.Printf("收到来自 %s 的消息，收件人是 %v\n", from, to)
	return h.SendResponse(250, "消息已接收")
}

func RunServer() {
	handlerFactory := func(conn net.Conn) smtp.Handler {
		return &MyHandler{smtp.NewDefaultHandler(conn)}
	}
	err := smtp.ListenAndServe(":2525", handlerFactory)
	if err != nil {
		log.Fatal(err)
	}
}
