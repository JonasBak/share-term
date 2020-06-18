package main

import (
	"fmt"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh/terminal"
	"io"
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

	disconnected, err := connect(ch, exit)
	if err != nil {
		panic(err)
	}

	spawnPty(chanWriter{Ch: ch})
	close(exit)
	select {
	case <-disconnected:
	case <-time.After(time.Second):
	}
}

func connect(ch chan []byte, exit chan struct{}) (chan struct{}, error) {
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/share"}
	fmt.Printf("Connecting to %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	disconnected := make(chan struct{})

	go func() {
		defer c.Close()
		for {
			select {
			case <-disconnected:
				return
			case b := <-ch:
				err := c.WriteMessage(websocket.TextMessage, b)
				if err != nil {
					fmt.Println("Failed to write to server:", err)
					return
				}
			case <-exit:
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					fmt.Println("Failed to close connection properly", err)
					return
				}
				select {
				case <-disconnected:
				case <-time.After(time.Second):
				}
				return
			}
		}
	}()

	go func() {
		defer close(disconnected)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				fmt.Println("Connection closed:", err)
				return
			}
			fmt.Printf("From server: %s\n", message)
		}
	}()

	return disconnected, nil
}

func spawnPty(writer chanWriter) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	c := exec.Command(shell)

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
				fmt.Printf("Error resizing pty: %s", err)
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

	// TODO check for resize
	writer.Ch <- []byte(fmt.Sprintf("DIM%d,%d", rows, cols))

	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, io.TeeReader(ptmx, &writer))

	return nil
}
