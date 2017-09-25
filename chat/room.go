package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"github.com/startDaemons/go-blueprints/trace"
	"github.com/stretchr/objx"
)

// Thoughts:
// - channels are just async message queues:
// 		use them to enable async logic
// - below, we use channels to enable thread safe logic:
// 		A single goroutine handles adding and removing
//		clients to/from the map.
//		The goroutine listens for messages on the join and 
//		leave chans
// - This also nicely separates the concerns into subsystems comprised 
//		of queues (chans) and listeners (goroutines)

type room struct {
	// forward holds incomming messages to be forwarded to 
	// other clients
	forward chan *message
	join chan *client
	leave chan *client
	clients map[*client]bool
	tracer trace.Tracer
}

func newRoom() *room {
	return &room{
		forward: make(chan *message),
		join: make(chan *client),
		leave: make(chan *client),
		clients: make(map[*client]bool),
		tracer: trace.Off(),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			// client joining
			r.clients[client] = true
			r.tracer.Trace("New client joined")
		case client := <-r.leave: 
			// client leaving
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("Client left")
		case msg := <-r.forward:
			// forward message to all clients
			r.tracer.Trace("Message recieved", msg.Message)
			for client := range r.clients {
				client.send <- msg
				r.tracer.Trace(" -- send to client")
			}
		}
	}
}

const (
	socketBufferSize = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{
	ReadBufferSize: socketBufferSize,
	WriteBufferSize: socketBufferSize,
}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	authCookie, err := req.Cookie("auth")
	if err != nil {
		log.Fatal("Failed to get auth cookie:", err)
		return
	}
	client := &client{
		socket: socket,
		send: make(chan *message, messageBufferSize),
		room: r,
		userData: objx.MustFromBase64(authCookie.Value),
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}