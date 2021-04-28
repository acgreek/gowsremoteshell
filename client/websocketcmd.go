package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

var (
	addr    = flag.String("addr", "localhost:9070", "http service address")
	cmdline = flag.String("cmdline", "/bin/sh", "the command line")
)

func handleConnection(cb *websocket.Conn) chan string {
	fmt.Printf("going into raw mode\n")
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	// Set stdin in raw mode.
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Printf("\n")
	}() // Best effort.
	output := make(chan string, 100)

	defer close(output)
	ctx, cancel := context.WithCancel(context.Background())

	// read from the websocket and output to stdout
	go func() {
		for {
			_, message, err := cb.ReadMessage()
			if err != nil {
				if errors.Is(err, io.EOF) {
					output <- "remote host request process or server closed socket"
				}
				output <- fmt.Sprintf("error reading from websocket: %s\n", err)
				cb.Close()
				cancel()
				return
			}
			_, err = os.Stdout.Write(message)
			if err != nil {
				panic(fmt.Sprintf("error reading from websocket: %s", err))
			}
		}
	}()

	reader := NewCancelRead(ctx, os.Stdin)
	// read from standard in and output the websocket
	for {
		buffer := make([]byte, 10, 100)
		n, err := reader.Read(buffer)
		if err != nil {
			output <- fmt.Sprintf("error reading from websocket: %s\n", err)
			cb.Close()
			break
		}
		buffer = buffer[:n]
		err = cb.WriteMessage(websocket.TextMessage, buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				output <- "remote host request process or server closed socket"
			}
			output <- fmt.Sprintf("error writing to the websocket: %s\n", err)
			cb.Close()
			break
		}
	}
	return output
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	u := url.URL{Scheme: "ws", Host: *addr, Path: fmt.Sprintf("v1/shell")}
	log.Printf("connecting to %s\n", u.String())
	dialer := websocket.DefaultDialer
	timeout := 1 * time.Minute
	ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)

	headers := make(http.Header)
	headers["CMD"] = []string{*cmdline}
	cb, _, err := dialer.DialContext(ctxTimeout, u.String(), headers)
	cancel()
	defer func() {
		err := cb.Close()
		if err != nil {
			log.Printf("error on close of websocket connection: %s", err)
		}
	}()
	if err != nil {
		panic(err)
	}
	fmt.Printf("connect\n")
	results := handleConnection(cb)
	c := 0
	for r := range results {
		fmt.Printf("%s", r)
		c++
	}
	fmt.Printf("done %d\n", c)
}
