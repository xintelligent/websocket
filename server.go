package websocket

import (
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"math/rand"
	"strconv"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

type Server struct {
	clients map[string]*Client
}

var _ io.Closer = (*Server)(nil)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewServer() *Server {
	return &Server{clients: make(map[string]*Client)}
}

func (s *Server) Register(key string, conn *websocket.Conn) *Client {
	client := &Client{
		wsConnect: conn,
		inChan:    make(chan []byte, 1024),
		outChan:   make(chan []byte, 1024),
		closeChan: make(chan struct{}, 1),
	}

	go client.readLoop()
	go client.writeLoop()
	if c, ok := s.clients[key]; ok {
		_ = c.Close()
	}
	s.clients[key] = client
	log.Printf("client %s is registered\n", key)

	return client
}

func (s *Server) RegisterWithKey(conn *websocket.Conn) (key string, client *Client) {
	key = RandKey()
	return key, s.Register(key, conn)
}

func (s *Server) Unregister(key string) {
	delete(s.clients, key)
	log.Printf("client %s is unregistered", key)
}

func (s *Server) Broadcast(data []byte) (err error) {
	var g errgroup.Group
	for _, c := range s.clients {
		g.Go(func(client *Client) func() error {
			return func() error {
				return client.WriteMessage(data)
			}
		}(c))
	}
	return g.Wait()
}

func (s *Server) BroadcastToOther(k string, data []byte) (err error) {
	var g errgroup.Group
	for key, client := range s.clients {
		if k == key {
			continue
		}
		g.Go(func(c *Client) func() error {
			return func() error {
				return c.WriteMessage(data)
			}
		}(client))
	}
	return g.Wait()
}

func (s *Server) Send(key string, data []byte) (err error) {
	if c := s.findClient(key); c != nil {
		return c.WriteMessage(data)
	}
	return ErrClientNotFound
}

func (s *Server) findClient(key string) (client *Client) {
	return s.clients[key]
}

func (s *Server) Close() (err error) {
	var g errgroup.Group
	for _, client := range s.clients {
		g.Go(func(c *Client) func() error {
			return c.Close
		}(client))
	}
	return g.Wait()
}

func RandKey() string {
	b := make([]rune, 8)
	l := len(letterRunes)
	for i := range b {
		b[i] = letterRunes[rand.Intn(l)]
	}
	return string(b) + "@" + strconv.Itoa(int(time.Now().UnixNano()/1e6))
}
