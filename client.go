package main

import (
	// "bytes"
	"fmt"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type chanWriter struct {
	Ch chan<- []byte
}

func (w *chanWriter) Write(p []byte) (int, error) {
	w.Ch <- append([]byte("TXT"), p...)
	return len(p), nil
}

func Client() {
	ch := make(chan []byte)
	exit := make(chan struct{})
	go connect(ch, exit)
	SpawnPty(chanWriter{Ch: ch})
	close(exit)
}

func connect(ch chan []byte, exit chan struct{}) {
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/share"}
	log.Printf("connecting to %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("from server: %s\n", message)
		}
	}()

	for {
		select {
		case <-done:
			return
		case b := <-ch:
			err := c.WriteMessage(websocket.TextMessage, b)
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-exit:
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func SpawnPty(writer chanWriter) error {
	c := exec.Command("bash")

	ptmx, err := pty.Start(c)
	if err != nil {
		return err
	}

	defer func() { _ = ptmx.Close() }()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH

	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }()

	rows, cols, err := pty.Getsize(os.Stdin)
	if err != nil {
		panic(err)
	}
	writer.Ch <- []byte(fmt.Sprintf("DIM%d,%d", rows, cols))

	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, io.TeeReader(ptmx, &writer))

	return nil
}
