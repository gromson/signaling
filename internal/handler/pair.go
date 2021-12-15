package handler

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type broadcast struct {
	client *client
	data   []byte
}

type pair struct {
	id        uuid.UUID
	clients   [2]*client
	broadcast chan *broadcast
	pairing   chan *client
	terminate chan struct{}
}

func newPair() (*pair, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create a pair")
	}

	return &pair{
		id:        id,
		clients:   [2]*client{nil, nil},
		broadcast: make(chan *broadcast),
		pairing:   make(chan *client),
		terminate: make(chan struct{}),
	}, nil
}

func (p *pair) run() {
	for {
		select {
		case c := <-p.pairing:
			err := pairClient(p, c)
			if err != nil {
				c.messageHandleErrors <- messageHandleError{
					Code: errorCodePairing,
					Desc: "Couldn't pair a client",
				}
			}

			c.setPair <- p
		case msg := <-p.broadcast:
			for _, c := range p.clients {
				if c != msg.client {
					c.incoming <- msg.data
				}
			}
		case <-p.terminate:
			for _, c := range p.clients {
				c.disconnect <- struct{}{}
			}
			return
		}
	}
}

func pairClient(p *pair, c *client) error {
	if p.clients[0] == nil {
		p.clients[0] = c
		return nil
	}

	if p.clients[1] != nil {
		return errors.New("the pair is already contain two clients")
	}

	p.clients[1] = c
	return nil
}
