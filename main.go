package main

import (
	"flag"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var server = flag.Bool("server", false, "run as server")

func main() {
	flag.Parse()
	if *server {
		Server()
	} else {
		Client()
	}
}
