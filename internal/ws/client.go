package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct{
	conn *websocket.Conn
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request)bool{
		return true
	},
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{conn: conn}
	hub.Register(client)
}