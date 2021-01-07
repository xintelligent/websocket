package main

import (
	ws "github.com/xintelligent/websocket"
	"log"
	"net/url"
	"time"
)

func main() {
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:8080", Path: "/ws"}

	client, resp, err := ws.NewClient(u, ws.ClientConfig{
		PingWait: time.Second,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("client %s is connected", resp.Header.Get("CLIENT_KEY"))

	for {
		data, err := client.ReadMessage()
		if err != nil {
			if err == ws.ErrClientIsClosed {
				log.Println("client is closed")
				return
			}
			log.Println("[error] recv error:", err)
			continue
		}
		log.Println("recv:", string(data))
	}
}
