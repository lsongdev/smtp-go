package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: smtp-go <command>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		RunServer()
	case "client":
		RunClient()
	default:
		fmt.Println("Usage: smtp-go <client|server>")
		os.Exit(1)
	}
}
