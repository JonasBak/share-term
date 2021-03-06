package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type session struct {
	dim [2]int
	w   []*websocket.Conn
	mux sync.Mutex
}

type serverState struct {
	s map[string]*session
}

func (s serverState) newSession() string {
	b := make([]byte, 4)
	rand.Read(b)
	name := fmt.Sprintf("%x", b)
	if _, taken := s.s[name]; taken {
		panic("TODO")
	}
	s.s[name] = &session{w: []*websocket.Conn{}}

	return name
}

var state = serverState{s: make(map[string]*session)}

var upgrader = websocket.Upgrader{}

var homeTemplate *template.Template
var notFoundTemplate *template.Template
var uiTemplate *template.Template

func share(w http.ResponseWriter, r *http.Request) {
	sessionName := state.newSession()

	log.WithFields(log.Fields{
		"session": sessionName,
	}).Info("Starting new session")

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"session": sessionName,
		}).Error("Failed to upgrade request")
		return
	}
	defer func() {
		log.WithFields(log.Fields{
			"session": sessionName,
		}).Info("Cleaning up after session")
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
		log.WithFields(log.Fields{
			"error":   err,
			"session": sessionName,
		}).Error("Failed to send message back to client")
		return
	}

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"session": sessionName,
			}).Warn("Channel closed")
			break
		}
		s := state.s[sessionName]
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
		s.mux.Lock()
		for i := range s.w {
			s.w[i].WriteMessage(websocket.TextMessage, message)
		}
		s.mux.Unlock()
	}
}

func Server() {
	log.SetLevel(log.DebugLevel)

	homeTemplate = template.Must(template.ParseFiles("./templates/base.html", "./templates/home.html"))
	notFoundTemplate = template.Must(template.ParseFiles("./templates/base.html", "./templates/not-found.html"))
	uiTemplate = template.Must(template.ParseFiles("./templates/base.html", "./templates/ui.html"))

	router := mux.NewRouter()
	router.HandleFunc("/share", share)
	router.HandleFunc("/sub/{session:[a-zA-Z0-9]+}", sub)
	router.HandleFunc("/s/{session:[a-zA-Z0-9]+}", webUI)
	router.HandleFunc("/", home)
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func sub(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sessionName := params["session"]

	if _, exists := state.s[sessionName]; !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"session": sessionName,
		}).Warn("Failed to upgrade watcher")
		return
	}

	s := state.s[sessionName]
	s.mux.Lock()
	s.w = append(s.w, c)
	s.mux.Unlock()

	c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("DIM%d,%d", s.dim[0], s.dim[1])))

	log.WithFields(log.Fields{
		"session": sessionName,
	}).Debug("Watcher joined")

	c.WriteMessage(websocket.TextMessage, []byte("TXTConnected\n"))

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if s, exists := state.s[sessionName]; exists {
				s.mux.Lock()
				// TODO use set for constant lookup
				for i := range s.w {
					if s.w[i] == c {
						s.w[i] = s.w[len(s.w)-1]
						s.w = s.w[:len(s.w)-1]
						break
					}
				}
				s.mux.Unlock()
			}
			log.WithFields(log.Fields{
				"session": sessionName,
				"status":  err,
			}).Debug("Watcher left")
			break
		}
	}
}

func webUI(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	session := params["session"]

	if _, exists := state.s[session]; !exists {
		w.WriteHeader(http.StatusNotFound)
		notFoundTemplate.Execute(w, session)
		return
	}

	uiTemplate.Execute(w, getScheme()+"://"+r.Host+"/sub/"+session)
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, r.Host)
}
