package main

import (
	"flag"
	"fmt"
	"os"
)

var addr = flag.String("addr", "", "for client: where to connect, for server: where to listen")
var server = flag.Bool("server", false, "run as server")
var insecure = flag.Bool("insecure", false, "run unencrypted (no https)")

func getScheme() string {
	if *insecure {
		return "ws"
	} else {
		return "wss"
	}
}

func main() {
	flag.Parse()
	if *addr == "" {
		if env_addr := os.Getenv("SHARE_TERM_ADDR"); env_addr != "" {
			*addr = env_addr
		} else {
			fmt.Println("Remote address must be set with either the -addr flag or the SHARE_TERM_ADDR env variable")
			os.Exit(1)
		}
	}
	if *server {
		Server()
	} else {
		Client()
	}
}
