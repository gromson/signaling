package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func TestHub_successful_pairing(t *testing.T) {
	h := newHub()

	t.Log("running a hub")
	go h.run()
	t.Log("the hub is running")

	conn1 := newWebSocketConnStub()
	conn2 := newWebSocketConnStub()

	apiClient1 := newApiClientStub(conn2)
	apiClient2 := newApiClientStub(conn1)

	client1 := newClient(conn1, h, apiClient1)
	client2 := newClient(conn2, h, apiClient2)

	t.Log("running clients")
	go client1.run()
	go client2.run()
	t.Log("the clients are running")

	resultChan1 := make(chan error)
	resultChan2 := make(chan error)
	go testExpectedCallInitConfirmationMessage(conn1, resultChan1)
	go testExpectedCallInitConfirmationMessage(conn2, resultChan2)

	t.Log("initializing a call")
	testInitializeCall(conn1)
	t.Log("the call initialization request has been sent")

	select {
	case err := <-resultChan1:
		if err != nil {
			t.Errorf("did not receive an expected call initialization confirmation: %s", err)
		}
	case err := <-resultChan2:
		if err != nil {
			t.Errorf("did not receive an expected answer confirmation: %s", err)
		}
	}

	if len(h.pairs) != 1 {
		t.Errorf("one pair expected in the hub")
	}
}

func testExpectedCallInitConfirmationMessage(conn *webSocketConnStub, result chan error) {
	timeout := time.Second
	timer := time.NewTimer(timeout)

	for {
		select {
		case wsMsg := <-conn.out:
			connMsg, err := newConnectionMessageFromBytes(wsMsg.data)
			if err != nil {
				result <- err
			}

			if connMsg.Typ == outgoingMessageCallInitialized || connMsg.Typ == outgoingMessageAnswerAccepted {
				close(result)
				return
			}
		case <-timer.C:
			result <- errors.Errorf("haven't got a call init confirmation in %v", timeout)
			return
		}
	}
}

func testInitializeCall(conn *webSocketConnStub) {
	msg := connectionMessage{
		Typ:     incomingMessageCall,
		Content: nil,
	}

	conn.in <- struct {
		typ  int
		data []byte
	}{typ: websocket.BinaryMessage, data: msg.Encode()}
}

type webSocketConnStub struct {
	in chan struct {
		typ  int
		data []byte
	}
	out chan struct {
		typ  int
		data []byte
	}
	isClosed bool
}

func newWebSocketConnStub() *webSocketConnStub {
	return &webSocketConnStub{
		in: make(chan struct {
			typ  int
			data []byte
		}),
		out: make(chan struct {
			typ  int
			data []byte
		}),
		isClosed: false,
	}
}

func (c *webSocketConnStub) ReadMessage() (messageType int, p []byte, err error) {
	if c.isClosed {
		return 0, nil, &websocket.CloseError{
			Code: websocket.CloseNormalClosure,
			Text: "Connection is closed",
		}
	}

	select {
	case msg := <-c.in:
		return msg.typ, msg.data, nil
	default:
		return 0, nil, nil
	}
}

func (c *webSocketConnStub) WriteMessage(messageType int, data []byte) error {
	if c.isClosed {
		return &websocket.CloseError{
			Code: websocket.CloseNormalClosure,
			Text: "Connection is closed",
		}
	}

	c.out <- struct {
		typ  int
		data []byte
	}{typ: messageType, data: data}

	return nil
}

func (c *webSocketConnStub) SetWriteDeadline(t time.Time) error {
	if c.isClosed {
		return &websocket.CloseError{
			Code: websocket.CloseNormalClosure,
			Text: "Connection is closed",
		}
	}
	return nil
}

func (c *webSocketConnStub) Close() error {
	c.isClosed = true
	return nil
}

type apiClientStub struct {
	peerWebSocketConn *webSocketConnStub
}

func newApiClientStub(conn *webSocketConnStub) *apiClientStub {
	return &apiClientStub{peerWebSocketConn: conn}
}

func (c *apiClientStub) Call(pairID uuid.UUID) error {
	msg := &connectionMessage{
		Typ:     incomingMessageAnswer,
		Content: pairID[:],
	}

	c.peerWebSocketConn.in <- struct {
		typ  int
		data []byte
	}{typ: websocket.TextMessage, data: msg.Encode()}

	return nil
}
