package websocket

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Client struct {
	wsConnect *websocket.Conn
	inChan    chan []byte
	outChan   chan []byte
	closeChan chan struct{}

	mutex    sync.Mutex
	isClosed bool
}

type ClientConfig struct {
	RequestHeader http.Header
	PingWait      time.Duration
}

func NewClient(u url.URL, config ClientConfig) (c *Client, resp *http.Response, err error) {
	var conn *websocket.Conn
	conn, resp, err = websocket.DefaultDialer.Dial(u.String(), config.RequestHeader)
	if err != nil {
		return
	}

	c = &Client{
		wsConnect: conn,
		inChan:    make(chan []byte, 1024),
		outChan:   make(chan []byte, 1024),
		closeChan: make(chan struct{}, 1),
	}

	go c.pingLoop(config.PingWait)
	go c.readLoop()
	go c.writeLoop()

	return
}

func (c *Client) ReadMessage() (data []byte, err error) {
	select {
	case data = <-c.inChan:
	case <-c.closeChan:
		err = ErrClientIsClosed
	}
	return
}

func (c *Client) WriteMessage(data []byte) (err error) {
	select {
	case c.outChan <- data:
	case <-c.closeChan:
		err = ErrClientIsClosed
	}
	return
}

func (c *Client) Close() (err error) {
	if err = c.wsConnect.Close(); err != nil {
		return
	}
	c.mutex.Lock()
	defer func() { c.mutex.Unlock() }()
	if !c.isClosed {
		close(c.closeChan)
		c.isClosed = true
	}
	return
}

func (c *Client) readLoop() {
	var (
		data []byte
		err  error
	)
	defer func() { _ = c.Close() }()
	for {
		if _, data, err = c.wsConnect.ReadMessage(); err != nil {
			return
		}
		select {
		case c.inChan <- data:
		case <-c.closeChan:
			return
		}
	}
}

func (c *Client) writeLoop() {
	var (
		data []byte
		err  error
	)
	defer func() { _ = c.Close() }()
	for {
		select {
		case data = <-c.outChan:
		case <-c.closeChan:
			return
		}
		if err = c.wsConnect.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}

func (c *Client) pingLoop(t time.Duration) {
	ticker := time.NewTicker(t)
	defer func() { ticker.Stop() }()
	var err error
	for {
		select {
		case <-ticker.C:
			if err = c.wsConnect.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("[error] send ping error:", err)
			}
		case <-c.closeChan:
			return
		}
	}
}
