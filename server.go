package main

import (
	// "fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var subs = []*websocket.Conn{}

var upgrader = websocket.Upgrader{} // use default options

func share(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	if err := c.WriteMessage(websocket.TextMessage, []byte("connected TODO url")); err != nil {
		log.Println("read:", err)
		return
	}
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		for i := range subs {
			subs[i].WriteMessage(websocket.TextMessage, message)
		}
		// fmt.Printf("%s", message)
	}
}

func Server() {
	log.SetFlags(0)
	http.HandleFunc("/share", share)
	http.HandleFunc("/sub", sub)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func sub(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	// defer c.Close()
	if err := c.WriteMessage(websocket.TextMessage, []byte("TXTconnected")); err != nil {
		log.Println("read:", err)
		return
	}
	subs = append(subs, c)
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/sub")
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/xterm/3.14.5/xterm.css" integrity="sha256-RogLy8bOb4a5F5dhZEi0c6gSwYTUgD7h8rfGj9ogn0c=" crossorigin="anonymous" />
<script type="text/javascript" src="https://cdnjs.cloudflare.com/ajax/libs/xterm/3.14.5/xterm.min.js"></script>
<script type="text/javascript">
window.addEventListener("load", function(evt) {
  var output = document.getElementById("output");
  var input = document.getElementById("input");
  var ws;
  var term = new Terminal();
  term.open(document.getElementById('terminal'));
  document.getElementById("open").onclick = function(evt) {
    if (ws) {
      return false;
    }
    ws = new WebSocket("{{.}}");
    ws.onopen = function(evt) {
    }
    ws.onclose = function(evt) {
      ws = null;
    }
    ws.onmessage = function(evt) {
      const type = evt.data.substring(0,3);
      switch(type) {
        case "TXT":
          term.write(evt.data.substring(3));
          break;
        case "DIM":
          const [rows, cols] = evt.data.substring(3).split(",");
          term.resize(parseInt(cols), parseInt(rows));
          break;
        default:
          console.err("could not parse message: " + evt.data);
      }
    }
    ws.onerror = function(evt) {
      alert("ERROR: " + evt.data);
    }
    return false;
  };
  document.getElementById("close").onclick = function(evt) {
    if (!ws) {
      return false;
    }
    ws.close();
    return false;
  };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<form>
<button id="open">Open</button>
<button id="close">Close</button>
</form>
</td><td valign="top" width="50%">
<div id="terminal"></div>
</td></tr></table>
</body>
</html>
`))
