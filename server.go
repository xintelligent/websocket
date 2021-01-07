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

// Server websocket服务器
type Server struct {
	clients map[string]*Client
}

var _ io.Closer = (*Server)(nil)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewServer 创建websocket服务器
func NewServer() *Server {
	return &Server{clients: make(map[string]*Client)}
}

// 注册客户端到服务器
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

// 注册未命名客户端到服务器
func (s *Server) RegisterWithKey(conn *websocket.Conn) (key string, client *Client) {
	key = RandKey()
	return key, s.Register(key, conn)
}

// 取消注册客户端
func (s *Server) Unregister(key string) {
	delete(s.clients, key)
	log.Printf("client %s is unregistered", key)
}

// 向所有客户端广播
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

// 向其他客户端广播
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

// 向指定客户端发送消息
func (s *Server) Send(key string, data []byte) (err error) {
	if c := s.findClient(key); c != nil {
		return c.WriteMessage(data)
	}
	return ErrClientNotFound
}

// 查找客户端
func (s *Server) findClient(key string) (client *Client) {
	return s.clients[key]
}

// 关闭服务器
func (s *Server) Close() (err error) {
	var g errgroup.Group
	for _, client := range s.clients {
		g.Go(func(c *Client) func() error {
			return c.Close
		}(client))
	}
	return g.Wait()
}

// 生成随机key
func RandKey() string {
	b := make([]rune, 8)
	l := len(letterRunes)
	for i := range b {
		b[i] = letterRunes[rand.Intn(l)]
	}
	return string(b) + "@" + strconv.Itoa(int(time.Now().UnixNano()/1e6))
}

// 获取全部客户端的key
func (s *Server) ClientKeys() (keys []string) {
	for k := range s.clients {
		keys = append(keys, k)
	}
	return
}

// 获取全部客户端
func (s *Server) Clients() (clients []*Client) {
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	return
}
