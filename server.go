package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type session struct {
	dim [2]int
	w   []*websocket.Conn
	mux sync.Mutex
}

type serverState struct {
	s map[string]*session
}

func (s serverState) newSession() (string, error) {
	name := "todo"
	if _, taken := s.s[name]; taken {
		panic("todo")
	}
	s.s[name] = &session{w: []*websocket.Conn{}}

	return name, nil
}

var state = serverState{s: make(map[string]*session)}

var upgrader = websocket.Upgrader{}

func share(w http.ResponseWriter, r *http.Request) {
	sessionName, err := state.newSession()
	if err != nil {
		log.Print("session start:", err)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func() {
		c.Close()
		s := state.s[sessionName]
		s.mux.Lock()

		for i := range s.w {
			s.w[i].WriteMessage(websocket.TextMessage, []byte("TXTDisconnected"))
			s.w[i].WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		}
		delete(state.s, sessionName)
		s.mux.Unlock()
	}()

	if err := c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("sharing terminal on %s/s/%s", r.Host, sessionName))); err != nil {
		log.Println("write:", err)
		return
	}

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		s := state.s[sessionName]
		s.mux.Lock()
		switch string(message[:3]) {
		case "DIM":
			parts := strings.Split(string(message[3:]), ",")
			cols, errC := strconv.Atoi(parts[0])
			rows, errR := strconv.Atoi(parts[1])
			if errC == nil && errR == nil {
				s.dim = [2]int{cols, rows}
			}
		default:
		}
		for i := range s.w {
			s.w[i].WriteMessage(websocket.TextMessage, message)
		}
		s.mux.Unlock()
	}
}

func Server() {
	log.SetFlags(0)
	router := mux.NewRouter()
	router.HandleFunc("/share", share)
	router.HandleFunc("/sub/{session:[a-z]+}", sub)
	router.HandleFunc("/s/{session:[a-z]+}", webUI)
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func sub(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sessionName := params["session"]

	if _, exists := state.s[sessionName]; !exists {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Session %s doesn't exist, TODO", sessionName)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	s := state.s[sessionName]
	s.mux.Lock()
	s.w = append(s.w, c)

	c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("DIM%d,%d", s.dim[0], s.dim[1])))

	s.mux.Unlock()

	c.WriteMessage(websocket.TextMessage, []byte("TXTConnected\n"))
}

func webUI(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	session := params["session"]

	if _, exists := state.s[session]; !exists {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Session %s doesn't exist, TODO", session)
		return
	}

	uiTemplate.Execute(w, "ws://"+r.Host+"/sub/"+session)
}

var uiTemplate = template.Must(template.ParseFiles("./templates/ui.html"))
