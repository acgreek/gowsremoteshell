package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	httpAddr = flag.String("coms", ":9070", "port to connect to to communicate with the endpoint")
)

func handleShell(w http.ResponseWriter, r *http.Request) {

	cmd := "/bin/sh"
	value, ok := r.Header["Cmd"]
	if ok {
		cmd = value[0]
	}
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - failed to upgrade connection to websocket: %s\n", err)))
		return
	}
	c := exec.Command(cmd)
	log.Printf("got shell request %d %s", cmd)
	f, err := pty.Start(c)
	if err != nil {
		panic(err)
	}
	go func() {
		defer f.Close()
		for {
			buf := make([]byte, 128)
			n, err := f.Read(buf)
			if err != nil {
				log.Printf("read error: shutting down: %s", err)
				conn.Close()
				break
			}
			if err != nil {
				log.Printf("shell error %sl", err)
				conn.Close()
				break
			}
			if n > 0 {
				buf = buf[:n]
				log.Printf("sending to client -%s- %X\n", buf, buf)
				w, err := conn.NextWriter(websocket.BinaryMessage)
				if err != nil {
					log.Println("write:", err)
					return
				}
				_, err = w.Write(buf)

				if err != nil {
					log.Println("write:", err)
					return
				}
				err = w.Close()
				if err != nil {
					log.Printf("failed to close writer: %s", err)
					return
				}
			}
		}
	}()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if err != nil {
			conn.Close()
			break
		}
		n, err := f.Write(message)
		log.Printf("sending to shell %d -%s- %X\n", n, message, message)
		if err != nil {
			conn.Close()
			break
		}
	}
}

func main() {
	flag.Parse()
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/v1/shell", handleShell)
	if err := http.ListenAndServe(*httpAddr, myRouter); err != nil {
		log.Printf("outbound connection listener failed")
	}
}
