package handler

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type pairInfo struct {
	client *client
	pairID uuid.UUID
}

type hub struct {
	pairs      map[uuid.UUID]*pair
	pair       chan *pairInfo
	register   chan *client
	unregister chan *pair
}

func newHub() *hub {
	return &hub{
		pairs:      make(map[uuid.UUID]*pair),
		pair:       make(chan *pairInfo),
		register:   make(chan *client),
		unregister: make(chan *pair),
	}
}

func (h *hub) run() {
	for {
		select {
		case clientPair := <-h.pair:
			p, err := h.findPair(clientPair.pairID)

			if err != nil {
				clientPair.client.messageHandleErrors <- messageHandleError{
					Code: errorCodePairID,
					Desc: "Couldn't find a peer to connect",
				}
			}

			if err == nil {
				p.pairing <- clientPair.client
			}
		case c := <-h.register:
			log.Debug("client registration request sent to the hub")
			p, err := newPair()
			if err != nil {
				log.WithError(err).Debug("couldn't register a client")
				c.messageHandleErrors <- messageHandleError{
					Code: errorCodeCall,
					Desc: "Couldn't find a peer to connect",
				}
			}

			if err == nil {
				go p.run()
				log.Debug("a new pair for registered client created and run")
				h.pairs[p.id] = p
				log.Debug("sending a client to a pair")
				p.pairing <- c
				log.Debug("client has been successfully sent to the pair")
			}
		case p := <-h.unregister:
			delete(h.pairs, p.id)
		}
	}
}

func (h *hub) findPair(pairID uuid.UUID) (*pair, error) {
	pair, ok := h.pairs[pairID]
	if !ok {
		return nil, errors.New("the pair with a given ID wasn't found in the hub")
	}

	return pair, nil
}
