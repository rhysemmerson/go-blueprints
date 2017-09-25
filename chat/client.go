package main

// Notes:
// notice that WriteMessage message and ReadMessage
// are blocking calls. client.read() and client.write
// translate these blocking calls into channel messages; 
// write() listens for messages on it's channel then 
// forwards them to WriteMessages()
// read() blocks until a ReadMessage() returns a value
// then sends it to the read channel

import (
	"time"
	"github.com/gorilla/websocket"
)

// client represent a single chatting user.
type client struct {
	socket *websocket.Conn
	// send is a channel on which messages are sent
	send chan *message
	room *room
	userData map[string]interface{}
}

func (c *client) read() {
	defer c.socket.Close()

	for {
		var msg *message
		err := c.socket.ReadJSON(&msg)
		if err != nil {
			return
		}
		msg.When = time.Now()
		msg.Name = c.userData["name"].(string)
		if avatarURL, ok := c.userData["avatar_url"]; ok {
			msg.AvatarURL = avatarURL.(string)
		}
		c.room.forward <- msg
	}
}

func (c *client) write() {
	defer c.socket.Close()

	for msg := range c.send {
		err := c.socket.WriteJSON(msg)
		if err != nil {
			return
		}
	}
}