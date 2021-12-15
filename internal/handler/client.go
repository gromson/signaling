package handler

import (
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 8) / 10
)

type webSocketConnection interface {
	io.Closer
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	SetWriteDeadline(t time.Time) error
}

type client struct {
	sync.WaitGroup
	conn                webSocketConnection
	api                 ApiClient
	hub                 *hub
	pair                *pair
	setPair             chan *pair
	setPairSuccess      chan struct{}
	incoming            chan []byte
	messageHandleErrors chan messageHandleError
	disconnect          chan struct{}
	terminate           chan struct{}
}

func newClient(conn webSocketConnection, hub *hub, apiClient ApiClient) *client {
	return &client{
		WaitGroup:           sync.WaitGroup{},
		conn:                conn,
		api:                 apiClient,
		hub:                 hub,
		setPair:             make(chan *pair),
		setPairSuccess:      make(chan struct{}),
		incoming:            make(chan []byte),
		messageHandleErrors: make(chan messageHandleError),
		disconnect:          make(chan struct{}),
		terminate:           make(chan struct{}),
	}
}

func (c *client) run() {
	c.Add(3)
	go c.handleWebSocketMessage()
	go c.handleIncomingMessage()
	go c.handleError()
	c.Wait()

	c.hub.unregister <- c.pair
}

func (c *client) handleWebSocketMessage() {
	defer func() {
		if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
			log.WithError(err).Error("error while writing closing message")
		}
		close(c.terminate)
		c.Done()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.WithError(err).Error("error while trying to read a message from the socket")

			var closeError *websocket.CloseError
			if errors.As(err, closeError) {
				return
			}
			continue
		}

		if message == nil {
			continue
		}

		if err := handleWebSocketRawMessage(c, message); err != nil {
			log.WithError(err).Error("error while handling incoming message")
		}
	}
}

func (c *client) handleIncomingMessage() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.Done()
	}()

	for {
		select {
		case data := <-c.incoming:
			msg := connectionMessage{
				Typ:     outgoingMessageSignaling,
				Content: data,
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, msg.Encode()); err != nil {
				log.WithError(err).Error("couldn't write message to the connection")
			}
		case p := <-c.setPair:
			c.pair = p
			c.setPairSuccess <- struct{}{}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.WithError(err).Error("error while setting write deadline")
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.WithError(err).Error("error on trying to ping")
			}
		case <-c.disconnect:
			if err := c.conn.Close(); err != nil {
				log.WithError(err).Error("error while trying to close a client's connection")
			}
		case <-c.terminate:
			return
		}
	}
}

func (c *client) handleError() {
	defer c.Done()

	for {
		select {
		case msgHandleError := <-c.messageHandleErrors:
			content, err := msgHandleError.Encode()
			if err != nil {
				log.WithError(err).Error("error while trying to encode handle error message")
			}

			msg := connectionMessage{
				Typ:     outgoingMessageError,
				Content: content,
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, msg.Encode()); err != nil {
				log.WithError(err).Error("couldn't write message to the connection")
			}
		case <-c.terminate:
			return
		}
	}
}

func handleWebSocketRawMessage(c *client, data []byte) error {
	incomingConnectionMessage, err := newConnectionMessageFromBytes(data)
	if err != nil {
		return errors.Wrap(err, "couldn't create message from raw data")
	}

	switch incomingConnectionMessage.Typ {
	case incomingMessageCall:
		c.hub.register <- c
		log.Debug("client sent to the hub")

		select {
		case <-c.setPairSuccess:
			if err := c.api.Call(c.pair.id); err != nil {
				c.messageHandleErrors <- messageHandleError{
					Code: errorCodeCall,
					Desc: "Couldn't initialized a call",
				}
				return errors.Wrap(err, "error response received from the API")
			}

			msg := connectionMessage{
				Typ:     outgoingMessageCallInitialized,
				Content: nil,
			}
			if err := c.conn.WriteMessage(websocket.BinaryMessage, msg.Encode()); err != nil {
				log.WithError(err).Error("couldn't write message to the connection")
			}
		}
	case incomingMessageAnswer:
		pairID, err := uuid.FromBytes(incomingConnectionMessage.Content)
		if err != nil {
			c.messageHandleErrors <- messageHandleError{
				Code: errorCodePairID,
				Desc: "Invalid pairID format",
			}
			return errors.Wrap(err, "invalid pairID format")
		}

		c.hub.pair <- &pairInfo{client: c, pairID: pairID}

		select {
		case <-c.setPairSuccess:
			msg := connectionMessage{
				Typ:     outgoingMessageAnswerAccepted,
				Content: nil,
			}
			if err := c.conn.WriteMessage(websocket.BinaryMessage, msg.Encode()); err != nil {
				log.WithError(err).Error("couldn't write message to the connection")
			}
		}
	case incomingMessageSignaling:
		c.pair.broadcast <- &broadcast{
			client: c,
			data:   incomingConnectionMessage.Content,
		}
	default:
		return errors.Errorf(
			"unknown message type received through the WebSocket: type %d with content: '%s'",
			incomingConnectionMessage.Typ,
			incomingConnectionMessage.Content)
	}

	return nil
}
