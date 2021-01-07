package main

import (
	"github.com/xintelligent/websocket"
	"log"
	"net/http"
)

var server = websocket.NewServer()

func main() {
	http.HandleFunc("/ws", websocketHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	key := websocket.RandKey()
	upgrader := websocket.DefaultUpgrader()
	conn, err := upgrader.Upgrade(w, r, http.Header{
		"CLIENT_KEY": []string{key},
	})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	client := server.Register(key, conn)
	defer func() { server.Unregister(key) }()
	for {
		data, err := client.ReadMessage()
		if err != nil {
			if err == websocket.ErrClientIsClosed {
				return
			}
		}
		log.Printf("[info] %s: %s\n", key, data)
		if err := server.BroadcastToOther(key, data); err != nil {
			log.Printf("[error] %s %s\n", key, err)
		}
	}
}
